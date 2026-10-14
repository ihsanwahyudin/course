import { AudioModule, RecordingPresets, createAudioPlayer, setAudioModeAsync, useAudioRecorder, useAudioRecorderState, type AudioPlayer } from 'expo-audio';
import * as Speech from 'expo-speech';
import { useCallback, useEffect, useRef, useState } from 'react';
import { Platform } from 'react-native';

import { API_URL, ApiError, api, authHeaders } from './api';

// ---- text to speech ----

// null = not known yet; false = the server has no voice (offline provider),
// so the device's own voice is used for the rest of the visit.
let serverVoice: boolean | null = null;
const webCache = new Map<string, Promise<string | null>>();
let current: { stop: () => void } | null = null;
// Bumped whenever playback starts or stops, so a stale queue knows to quit.
let generation = 0;

const ttsUrl = (text: string) => `${API_URL}/api/v1/tts?text=${encodeURIComponent(text)}`;
const pause = (ms: number) => new Promise((r) => setTimeout(r, ms));

// Web: the sentence's audio as a blob URL (fetched once), or null without a server voice.
function webAudioUrl(text: string): Promise<string | null> {
  let p = webCache.get(text);
  if (!p) {
    p = fetch(ttsUrl(text), { credentials: 'include', headers: authHeaders() })
      .then(async (res) => {
        if (!res.ok) {
          if (res.status === 501) serverVoice = false;
          webCache.delete(text);
          return null;
        }
        serverVoice = true;
        return URL.createObjectURL(await res.blob());
      })
      .catch(() => {
        webCache.delete(text);
        return null;
      });
    webCache.set(text, p);
  }
  return p;
}

// Start fetching a sentence of a queue before its turn.
function prefetch(text: string) {
  if (Platform.OS === 'web' && serverVoice !== false) void webAudioUrl(text);
}

function deviceVoice(text: string, onDone: () => void) {
  Speech.speak(text, { language: 'en-US', rate: 0.95, onDone, onStopped: onDone, onError: onDone });
  current = { stop: () => void Speech.stop() };
}

async function serverAudio(text: string, onDone: () => void, gen: number): Promise<boolean> {
  if (serverVoice === false) return false;
  if (Platform.OS === 'web') {
    const url = await webAudioUrl(text);
    if (!url) return false;
    if (gen !== generation) return (onDone(), true);
    const el = new window.Audio(url);
    el.onended = onDone;
    el.onerror = onDone;
    current = { stop: () => (el.pause(), onDone()) };
    await el.play();
    return true;
  }
  if (serverVoice === null) {
    // Ask once whether the server can speak before handing URLs to the player.
    const res = await fetch(ttsUrl(text), { headers: authHeaders() });
    serverVoice = res.ok;
    if (!res.ok) return false;
  }
  if (gen !== generation) return (onDone(), true);
  const player: AudioPlayer = createAudioPlayer({ uri: ttsUrl(text), headers: authHeaders() });
  const sub = player.addListener('playbackStatusUpdate', (st) => {
    if (st.didJustFinish) {
      sub.remove();
      player.remove();
      onDone();
    }
  });
  current = {
    stop: () => {
      sub.remove();
      player.remove();
      onDone();
    },
  };
  player.play();
  return true;
}

// Plays one sentence; resolves when it ends or is stopped.
function playOnce(text: string, gen: number): Promise<void> {
  return new Promise((resolve) => {
    let settled = false;
    const done = () => {
      if (!settled) {
        settled = true;
        resolve();
      }
    };
    serverAudio(text, done, gen)
      .then((ok) => {
        if (!ok && gen === generation) deviceVoice(text, done);
        else if (!ok) done();
      })
      .catch(() => (gen === generation ? deviceVoice(text, done) : done()));
  });
}

function stopAll() {
  generation++;
  current?.stop();
  current = null;
}

type Active = { key: string; index: number; gen: number };

/**
 * Plays English text: the server's voice when available, else the device's.
 * `speakAll` reads a list one item at a time and reports which one is
 * playing (`index`), so the UI can highlight it.
 */
export function useSpeak() {
  const [active, setActive] = useState<Active | null>(null);
  const mounted = useRef(true);
  useEffect(
    () => () => {
      mounted.current = false;
    },
    [],
  );
  const run = useCallback(async (texts: string[], key: string) => {
    stopAll();
    const gen = ++generation;
    const clear = () => mounted.current && setActive((a) => (a?.gen === gen ? null : a));
    // Fetch the whole list up front so the items follow each other without gaps.
    texts.slice(1).forEach(prefetch);
    for (let i = 0; i < texts.length; i++) {
      if (gen !== generation) return clear();
      if (mounted.current) setActive({ key, index: i, gen });
      await playOnce(texts[i], gen);
      if (i + 1 < texts.length && gen === generation) await pause(300);
    }
    clear();
  }, []);
  const speak = useCallback((text: string) => run([text], text), [run]);
  const speakAll = useCallback((texts: string[]) => run(texts, texts.join('\n')), [run]);
  const stop = useCallback(() => {
    stopAll();
    setActive(null);
  }, []);
  return { speak, speakAll, stop, speaking: active?.key ?? null, index: active?.index ?? null };
}

// ---- recording + transcription ----

const MAX_MS = 60_000;

const typeFor = (uri: string, blobType: string) => {
  if (blobType) return blobType;
  if (/\.(m4a|mp4)$/i.test(uri)) return 'audio/mp4';
  if (/\.webm$/i.test(uri)) return 'audio/webm';
  if (/\.wav$/i.test(uri)) return 'audio/wav';
  return 'audio/mp4';
};

/** Records a short spoken answer and turns it into text on the server. */
export function useRecorder(onText: (text: string) => void) {
  const recorder = useAudioRecorder(RecordingPresets.HIGH_QUALITY);
  const state = useAudioRecorderState(recorder, 250);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(
    () => () => {
      if (timer.current) clearTimeout(timer.current);
    },
    [],
  );

  async function stop() {
    if (timer.current) clearTimeout(timer.current);
    if (!recorder.isRecording) return;
    setBusy(true);
    try {
      await recorder.stop();
      const uri = recorder.uri;
      if (!uri) throw new Error('no recording');
      const blob = await (await fetch(uri)).blob();
      const { text } = await api.transcribe(blob, typeFor(uri, blob.type));
      if (text.trim()) onText(text.trim());
    } catch (e) {
      setError(e instanceof ApiError ? e.code : 'record_failed');
    } finally {
      setBusy(false);
    }
  }

  async function start() {
    setError(null);
    try {
      const perm = await AudioModule.requestRecordingPermissionsAsync();
      if (!perm.granted) {
        setError('mic_denied');
        return;
      }
      await setAudioModeAsync({ playsInSilentMode: true, allowsRecording: true });
      await recorder.prepareToRecordAsync();
      recorder.record();
      // Keep recordings within the server's size limit.
      timer.current = setTimeout(() => void stop(), MAX_MS);
    } catch {
      setError('mic_unavailable');
    }
  }

  return { start, stop, recording: state.isRecording, durationMs: state.durationMillis, busy, error };
}
