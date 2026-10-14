import { AudioModule, RecordingPresets, createAudioPlayer, setAudioModeAsync, useAudioRecorder, useAudioRecorderState, type AudioPlayer } from 'expo-audio';
import * as Speech from 'expo-speech';
import { useCallback, useEffect, useRef, useState } from 'react';
import { Platform } from 'react-native';

import { API_URL, ApiError, api, authHeaders } from './api';

// ---- text to speech ----

// null = not known yet; false = the server has no voice (offline provider),
// so the device's own voice is used for the rest of the visit.
let serverVoice: boolean | null = null;
const webCache = new Map<string, string>();
let current: { stop: () => void } | null = null;

const ttsUrl = (text: string) => `${API_URL}/api/v1/tts?text=${encodeURIComponent(text)}`;

function deviceVoice(text: string, onDone: () => void) {
  Speech.speak(text, { language: 'en-US', rate: 0.95, onDone, onStopped: onDone, onError: onDone });
  current = { stop: () => void Speech.stop() };
}

async function serverAudio(text: string, onDone: () => void): Promise<boolean> {
  if (serverVoice === false) return false;
  if (Platform.OS === 'web') {
    let url = webCache.get(text);
    if (!url) {
      const res = await fetch(ttsUrl(text), { credentials: 'include', headers: authHeaders() });
      if (!res.ok) {
        if (res.status === 501) serverVoice = false;
        return false;
      }
      url = URL.createObjectURL(await res.blob());
      webCache.set(text, url);
    }
    serverVoice = true;
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

/** Plays English text: the server's voice when available, else the device's. */
export function useSpeak() {
  const [speaking, setSpeaking] = useState<string | null>(null);
  const mounted = useRef(true);
  useEffect(
    () => () => {
      mounted.current = false;
    },
    [],
  );
  const speak = useCallback(async (text: string) => {
    current?.stop();
    current = null;
    const done = () => mounted.current && setSpeaking((s) => (s === text ? null : s));
    setSpeaking(text);
    try {
      if (await serverAudio(text, done)) return;
    } catch {
      // fall through to the device voice
    }
    deviceVoice(text, done);
  }, []);
  const stop = useCallback(() => {
    current?.stop();
    current = null;
    setSpeaking(null);
  }, []);
  return { speak, stop, speaking };
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
