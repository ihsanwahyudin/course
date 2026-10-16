# Lesson content guide

The standard every FluentPath lesson follows, for whoever writes or edits course content (people and AI agents alike).
FluentPath teaches English to **Indonesian adults, starting from zero**, in English and Indonesian UI.

The Go validator enforces the structure (the API refuses to start otherwise). This guide covers that, and the things a
validator can't check: whether a total beginner will actually understand.

## 1. Who we write for

- A **super-beginner** with no grammar background, reading the **Indonesian** UI. If a sentence needs a grammar term to make sense,
  it needs rewriting or the term needs explaining.
- An adult: real situations (office, warung, ojek, kos, hospital, bank, train station), not children's topics.
- Indonesian contexts: names like Rina, Budi, Sari, Dimas, Pak Hasan, Bu Ani; places like Jakarta, Bandung, Surabaya; prices in
  rupiah; phone numbers starting 08…

## 2. Clarity rules (the "would a beginner get confused?" test)

A learner read *"**an** before a vowel sound … **a university**"* and asked: "u is a vowel, so why *a* university? And h is a
consonant, so why *an* hour?" The rule was right, but the example seemed to break it and nothing said why. Every lesson must avoid
that. The model fix is the material section `sound-not-letter` in `english/lessons/a1-articles-demonstratives.json`.

1. **Never show an example that seems to break the rule without saying why.** Explain it right there, in plain words:
   *a university* (dibaca "yu-ni-ver-si-ti": bunyi pertama "y", konsonan), *an hour* (h tidak dibaca), *he has* (not *haves*),
   *children / feet / people*, *goes / watches* (+es).
2. **Say when something is an exception** and that it just has to be remembered; give the short list.
3. **Explain jargon the first time it appears in a lesson**, in plain Indonesian, or avoid it: vowel/consonant sound, syllable,
   subject, object, verb, noun, adjective, auxiliary, contraction, countable/uncountable, plural, possessive, -ing form, intonation.
   Example: "**subjek** = siapa yang melakukan, misalnya *I, Rina, my mother*".
4. **No bare IPA.** Show a phonetic symbol only next to an everyday spelling of the sound: "/iː/ seperti 'ii' panjang di *see*".
5. **Name the Indonesian trap.** Where learners translate word by word, say so explicitly:
   - no *to be* in Indonesian (*Saya guru* → *I **am** a teacher*);
   - no plural marking (*dua buku* → *two book**s***);
   - one *dia* for *he* and *she*;
   - adjective before the noun (*tas hitam* → *a **black** bag*);
   - no tense marking (*kemarin saya di Bandung* → *I **was** in Bandung yesterday*);
   - *punya* = *have/has*; *sedang* = *be + -ing*;
   - time: *setengah tujuh* = **6:30** = *half past six* (not seven).
6. **Flag British vs American** where both are common: flat/apartment, at/on the weekend, lift/elevator, zed/zee, -our/-or.
7. **Explanations say why**, not just "correct": every question's `explanation` names the rule that makes the answer right
   (and, where useful, why the tempting wrong option is wrong).
8. **Natural Indonesian** with "kamu", short sentences, friendly tone. English examples stay English.
9. **Stay in scope.** Explain what the lesson teaches; don't teach grammar from later lessons to explain it. If the learner
   meets a form early (a fixed chunk like *I'd like*), call it a phrase to remember.

Where a "why" needs space, put it in the **reading material** (not read aloud, so it can be as long as it needs): a short
explanation, a table of words with how they're said, the trick to remember, and a check question. Keep step bodies short (aim
≤ 600 characters) but give the key "why" there too.

## 3. Level and grammar scope

- A lesson may only use grammar the learner has already met: every A0 lesson plus the lessons in its prerequisite chain
  (see the node's `prerequisiteIds` in `<level>.json`).
- A1: no past simple (except *was/were* in its own lesson), no future forms, no perfect tenses, no comparatives.
- A2: A0/A1 grammar plus the A2 grammar of the node's prerequisite chain only (no past simple without
  `a2-past-simple-regular` in the chain, no *will* without `a2-will`, no present perfect before `a2-present-perfect-intro`).
- Tense lessons (past, future) show when the action happens with a `timeline` step where it helps.
- The node's `objectives` and their ids are fixed: teach and test exactly those. Use its `commonErrors` in teaching notes, tips
  and distractors.

## 4. Files

Each ready node has two files (see `english/lessons/a1-to-be.json` and `english/plans/a1-to-be.json`):

- **`lessons/<nodeId>.json`** — `intro`; `teachingNotes` (for the AI tutor: what to stress, the common errors, what NOT to teach
  yet); `keyPoints` (one per objective); `guidedPractice` (3–5) and `independentPractice` (2–3); `assessment` (the hand-written
  checkpoint, 5–7 items); `material` (3–5 sections of `text`, `tip`, `examples`, `check` blocks); `suggestedQuestions`;
  `flashcards` (5–8); `resources` (0–3 links). Every text a learner sees has its `i18n.id` translation (the validator checks):
  `i18n.id.assessment` (item id → Indonesian prompt), key point `i18n.id.title`/`explanation`, example-row `i18n.id.note`, and
  check-block `i18n.id.prompt` (the options stay English: they're what's being learnt; a bare gap sentence gets an instruction,
  "Lengkapi: I need ___ egg.").
- **`plans/<nodeId>.json`** — the hand-written steps, `{"en": Plan, "id": Plan}`. 8–12 steps (concept, examples, question,
  timeline, flashcards, artifact = cheat sheet, resources last). First step is a concept; teach before testing; every objective
  has a question; 3–5 questions.

Markup in course text: `**bold**`, `*italic*`, `` `code` ``, line breaks, and lines starting with `- ` (shown as bullets).

## 5. Questions

- **Test the taught skill, not input formatting.** Each prompt has one natural situation and one task. A0 mostly taps; allow at most
  one text item per lesson (two in a level checkpoint), restricted to a taught word or fixed phrase of at most three words. A1 mostly
  taps, with at most two text items per lesson (three in a level checkpoint), on short taught sentence patterns. These limits apply to
  authored fallback content as well as AI-written questions. A2+ retains productive writing/speaking tasks.
- **Choose the interaction by the task:** `mcq` for one right option; `multi` for all the right items (3–6 options, at least one right
  and one wrong); `match` for 2–6 distinct pairs with one clear partner each; `letters` for spelling one word; `word_order` for a short
  taught sentence with one natural order. Never ask beginners to type a list or to put dashes between spelled letters.
- **Letter tiles:** the prompt never displays the answer word; play its spelling or give its meaning. No decoy letters at A0; at most
  two at A1. Repeated letters are separate tiles. Listening to a new name/word's spelled letters tests letter recognition, not knowing
  that word's meaning. Do not ask for an untaught translation as an indirect way of testing spelling.
- **Typed items** use `acceptableAnswers` and a rubric; tapped items use `type` and the corresponding `options`/`correctOption`,
  `correctOptions`, `pairs`, or `solution`/`tokens`. Both languages have the same options, pairs and keys; translate the instruction.
- **Fair grading:** equivalent list separators (spaces, commas, `and`, `dan`) and case do not change a content answer. If capitalization,
  punctuation or an exact format is itself the objective, state that clearly in the learner-visible prompt and assess it. A hidden
  rubric or uppercase reference answer must not create a requirement the learner was never told about. Never globally strip conjunctions from
  sentences or ignore a format being tested. Keep correctly formatted reference answers for genuine writing/editing tasks.
- **Teaching before testing:** make vocabulary and patterns explicit in the material before asking learners to use them. Word tiles
  supply words but do not teach their meanings. Simplify rules for beginners by naming the particular pattern being practised, without
  presenting a pattern such as *noun + please* as the only possible position of *please* in English.
- **A complete question:** a response-length instruction such as “Tuliskan 1 huruf yang benar” does not identify an answer. Supply
  a clue, situation, source sentence, or listening audio; the hidden key and rubric are not learner-visible clues. AI checkpoints
  require a verdict for every question; unavailable or incomplete verification falls back to authored questions.
- **One clear, correct answer.** Every multiple-choice distractor must be clearly wrong *for this exact sentence* (asking for "a
  vowel" with two vowels among the options, or offering *and* where *and* is also grammatical, are broken questions). Use real
  mistakes Indonesian learners make as distractors.
- **Word order:** a natural sentence of 3–9 words, tokens shuffled, never letters to spell; pick sentences with only one natural
  order.
- **No giveaways:** nothing shown before a question contains its answer sentence; practice questions never reuse checkpoint
  items or their exact answers.
- **Listening questions** put the English the learner hears in `listen` and never in the prompt.
  `listen` is read by one voice (at most 300 characters): in a dialogue, the speaker labels ("Rina: …") are read aloud too.
- **Length:** a step question's prompt is at most 400 characters; a checkpoint item's may be up to 800, so it can quote a
  short email, message or notice to read.
- **Text checkpoint items** have short, checkable `acceptableAnswers` with the common
  valid variants: contractions both ways, keeping the apostrophe, which matching keeps too (*didn't* and *did not*, never only *didnt*), with/without article, "9 45"/"0945" for times, "22 000" for prices) plus a `rubric` for
  partial credit (at most 20 accepted answers: accents don't matter, so *cafe*/*café* is one; leave rarer phrasings to the
  rubric and the AI grader). Keep case/punctuation in references where those are assessed. Authored items of every type are the fallback
  when the AI can't write a checkpoint and the examples the AI matches in level and style,
  so make them exemplary.
- The checkpoint the learner normally sees is **written by the AI for every attempt** from the lesson (see the API README); the
  pass mark is applied on the server.

## 6. Both languages

- `en` and `id` plans have the same steps, kinds and identical questions (type, objective, options in the same order, answer,
  tokens, solution, listen). Only wording differs: an Indonesian instruction in `id` is fine.
- The `id` field of word tiles and examples, and flashcard backs, hold the Indonesian meaning in **both** versions.

## 7. Resources

Only links you have opened and checked: YouTube via `https://www.youtube.com/oembed?url=<video url>&format=json` (title and
channel must match the topic), pages by fetching them. Prefer British Council LearnEnglish, BBC Learning English, Cambridge,
engVid, Woodward English. Say in the note when a link is above the lesson's level, for kids/teens, or covers extra topics.

## 8. Audio

- Every English sentence a plan reads aloud (word tiles, examples, cheat-sheet examples, `listen`) and every checkpoint `listen` is
  recorded once and committed in `<course>/audio/`.
- New lessons go into `audio/pending.json`. **Audio is made only after the content is reviewed**: remove the node from
  `pending.json`, then `cd apps/api && go run ./cmd/server bundle-audio` (only new or changed sentences are recorded).
- Changing a spoken English sentence means re-recording it. Prefer changing explanations and Indonesian glosses.

## 9. Editing a lesson that learners are already taking

- **Wording only** (explanations, titles, prompts, Indonesian glosses, material): applied in place; learners keep their step.
- **Steps or questions changed** (count, kinds, options, answers, tokens, solutions, listen): learners on the old version start the
  new one (the old session is kept).
- `material` is not part of the steps and can always be improved freely.

## 10. Validate before committing

```bash
cd apps/api && go run ./cmd/coursecheck                     # every course, as the API checks it at startup
cd apps/api && go test ./internal/tutor -run TestAuthoredPlans
scripts/e2e.sh                                               # every ready lesson mastered end to end (mock AI)
python3 scripts/checkpoint_audit.py --output review/checkpoint-audit-candidates.json
```

Then re-read every question and checkpoint item yourself: is the answer right and the only one? Would a beginner understand the
explanation in Indonesian?

The audit script inventories every assessment and EN/ID step question. Its vocabulary and wording flags require contextual human
review; an inflected form, proper name, quoted reading passage or a spelled word can be legitimate. Record reviewed changes and
decisions in `review/checkpoint-audit-id.md`. A successful structural validator alone does not prove sound teaching.

For OpenAI verification with no TTS, explicitly set both `LIVE_SPEECH=off` and `WARM_SPEECH=off`; disable checkpoint prefetching for
small controlled runs. New audio remains pending owner review. Test a fresh checkpoint, inspect its saved source, and distinguish
AI output from the authored fallback. Never delete learner progress merely to clear an old checkpoint.
