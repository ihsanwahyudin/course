# TOEFL iBT Prep (`toefl-ibt`): research notes

Roadmap-only track: every node is `"contentStatus": "coming_soon"`. Node prefix: `ibt-`.

## The 2026 format change

ETS announced the change on 29 May 2025. The updated TOEFL iBT has been used for every test taken **on or after 21 January 2026**. This track follows only the updated format. Do not reuse old material: the Integrated Writing task, the four old Speaking tasks, long reading passages and the 0–120 total as the main score are all gone.

Main changes:
- Reading and Listening use a **two-stage (multistage) adaptive** design: a "router" module, then a lower or upper module. Writing and Speaking are linear, so everyone on a form gets the same tasks.
- New task types throughout (table below).
- Scores are reported on a **1–6 band scale** in half-band steps, aligned to the CEFR.
- ETS promises scores "within 72 hours or less".

## Format (2026 Update Test Blueprint and Specifications)

| Section | Task types (official names) | Items | Estimated time |
|---|---|---|---|
| Reading (adaptive) | Complete the Words (30, B1–C1+) · Read in Daily Life (5–15, A1–C1) · Read an Academic Passage (5–15, B1–C2) | 50 (35 raw points; includes unscored pretest items) | router 18–21 min + module 9 min |
| Listening (adaptive) | Listen and Choose a Response (15–19, A1–B2) · Listen to a Conversation (10, A2–C1) · Listen to an Announcement (6–10, A2–C1) · Listen to an Academic Talk (8–16, A2–C2) | 47 (35 raw points; includes unscored pretest items) | router 18 min + module 7 min (lower) / 11 min (upper) |
| Writing (linear) | Build a Sentence (10) · Write an Email (1) · Write for an Academic Discussion (1) | 12 | 23 min |
| Speaking (linear) | Listen and Repeat (7) · Take an Interview (4) | 11 | 8 min |

Task details from the blueprint and the ETS section pages:
- **Complete the Words**: fill in partially missing words in a short academic text. Measures vocabulary, word form and meaning from context.
- **Read in Daily Life**: short everyday texts such as a sign, menu, email or social media post, in 2- or 3-item sets. Texts run from about 15–50 words.
- **Read an Academic Passage**: a university-style passage of up to about 200 words. Covers main idea, details, inference and text structure.
- **Listening**: audio is played once, and note-taking is allowed. Inputs run from very short utterances to talks of up to about 250 words. Accents include US/Canadian, British and Australian. Some audio is AI-generated.
- **Build a Sentence**: one intact sentence, then a reply with its words or phrases scrambled. The test taker reorders them into a grammatical reply.
- **Write an Email**: an academic or social situation, such as making a request, giving information or proposing a solution.
- **Write for an Academic Discussion**: state and support an opinion in an online class discussion and respond to classmates' posts.
- **Listen and Repeat**: repeat 7 sentences from a campus or academic scenario. Scored on accuracy and intelligibility.
- **Take an Interview**: answer 4 questions about experiences and opinions. Scored on elaboration, grammar, vocabulary and prosody.

Total time: the blueprint gives roughly **1 hr 23 min to 1 hr 29 min** of testing time. The extracted text is garbled and the section times add up to about 83–90 min. The ETS content page says to "allow approximately two hours", which presumably includes directions and check-in. There are no scheduled breaks.

Raw points: Reading and Listening items are worth 1 each, Build a Sentence 1 each, and Email, Academic Discussion, Listen and Repeat and Interview 5 each.

Scoring method: Reading and Listening are machine-scored. Writing and Speaking are scored by ETS's AI engine. ETS says human raters remain part of the process, and the engines are trained on human ratings.

Delivery: computer only, either at a test centre or through the **Home Edition** with a live proctor. The Paper Edition ended after 20 January 2024. Scores are valid for 2 years.

## Scoring and official CEFR mapping

Each section is scored 1–6 in half bands. The overall band is the **average of the four section bands, rounded to the nearest half band**. For example, 5.125 rounds to 5 and 5.25 rounds to 5.5. The 0–120 total, by contrast, was a sum.

| Band | CEFR | Comparable 0–120 total |
|---|---|---|
| 6 | C2 | 114+ |
| 5.5 / 5 | C1 | 107+ / 95+ |
| 4.5 / 4 | B2 | 86+ / 72+ |
| 3.5 / 3 | B1 | 58+ / 44+ |
| 2.5 / 2 | A2 | 34+ / 24+ |
| 1.5 / 1 | A1 | 12+ / 0+ |

From January 2026 to January 2028, score reports also show a **comparable 0–120 total**, the midpoint of the equivalent range. ETS publishes per-section 0–30 equivalents on its score-scale page. Many institutions still state requirements in 0–120, so the `ibt-score-scale` node teaches converting between the two scales.

## Who takes it (Indonesia)

TOEFL iBT is the TOEFL taken for **study abroad** and for international scholarships and admissions. It is different from TOEFL ITP, the paper-based institutional test that is common at Indonesian universities and is generally accepted only domestically.

Media reports on LPDP 2026 give minimums of **TOEFL iBT 80 for overseas master's and 94 for overseas doctoral** study. ITP is listed for domestic study only. These figures come from secondary sources, not an official LPDP document, so check lpdp.kemenkeu.go.id before quoting them in product copy. An 80 on the 0–120 scale falls in band 4 (72–85), and 94 falls in band 4.5 (86–94).

ETS advertises test centres and the Home Edition worldwide. I could not verify current Indonesian test-centre locations or the fee from an official ETS page. A reseller lists about IDR 3.2 million, which is unverified.

## How the track maps to the test

The track's levels are preparation stages, not CEFR levels. `recommendedAfter` is English **B1**. The test spans A1–C2, but most study-abroad targets are band 4–5 (B2–C1), so learners should start with B1 grammar and keep working on the English B2 and C1 nodes in parallel.

| Level | Nodes (24 total) |
|---|---|
| `Start` – Format & Strategy | The 2026 TOEFL iBT at a Glance · Scores, Bands & CEFR · Timing & Test-Day Strategy · Academic & Campus Vocabulary |
| `Reading` | Complete the Words · Read in Daily Life · Read an Academic Passage · Skimming, Scanning & Paraphrase |
| `Listening` | Listen and Choose a Response · Listen to a Conversation · Listen to an Announcement · Note-Taking for Listening · Listen to an Academic Talk |
| `Writing` | Build a Sentence · Write an Email · Write for an Academic Discussion · Self-Editing for Accuracy |
| `Speaking` | Intelligibility: Stress, Rhythm & Intonation · Listen and Repeat · Take an Interview · Organised Answers Under Time Pressure |
| `Mock` – Full Practice Tests | Timed Section Practice · Full Practice Test · Score Review & Study Plan |

Practice bands produced in the app are **estimates**, not ETS scores. The tutor policies say so explicitly.

## Uncertain or not verified

- **Per-task timing**: response time for Take an Interview and minutes for Write an Email and Academic Discussion are not on the ETS pages I could read. The teacher FAQ PDF could not be text-extracted. Nodes therefore avoid exact per-task timings.
- **Item counts**: older ETS pages give ranges such as 35–48 Reading and 35–45 Listening items. The 2026 blueprint and the current content page give 50 and 47. I used the latter. The blueprint says specifications were "subject to minor revisions" before launch.
- **Navigation**: I did not find whether test takers can return to earlier questions within a module.
- **Indonesian figures**: LPDP thresholds, fees and test-centre list are not confirmed from official sources (see above).

## Sources (accessed 15 October 2026)

- ETS, TOEFL iBT Test Content: https://www.ets.org/toefl/test-takers/ibt/about/content.html
- ETS, Reading / Listening / Writing / Speaking section pages: https://www.ets.org/toefl/test-takers/ibt/about/content/reading.html, …/listening.html, …/writing.html, …/speaking.html
- ETS, *TOEFL iBT Test: 2026 Update Test Blueprint and Specifications Document* (PDF): https://www.in.ets.org/content/dam/ets-org/pdfs/toefl/toefl-ibt-test-specifications-2026.pdf
- ETS, TOEFL iBT Score Scale Update (1–6, CEFR, 0–120 comparison): https://www.ets.org/toefl/institutions/ibt/score-scale-update.html
- ETS press release, TOEFL Transformation Announcement (29 May 2025): https://www.ets.org/news/press-releases/toefl-transformation-announcement.html
- ETS, Upcoming updates (Home Edition and test centres, eligibility): https://www.ets.org/toefl/test-takers/ibt/upcoming-updates-jan-2026.html
- ETS, TOEFL iBT Paper Edition (discontinued after 20 Jan 2024): https://www.ets.org/toefl/test-takers/ibt/about/content/paper.html
- Secondary (seen via search results, unofficial), LPDP thresholds: https://ielts.idp.com/indonesia/about/news-and-articles/article-minimum-toefl-pte-ielts-requirement-lpdp-scholarship and https://www.rctiplus.com/news/detail/gaya-hidup/5379382/berapa-skor-toefl-untuk-beasiswa-lpdp-2026-
