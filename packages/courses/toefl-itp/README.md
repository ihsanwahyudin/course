# TOEFL ITP Prep (`toefl-itp`): research notes

Roadmap-only track (every lesson is `coming_soon`). It targets **TOEFL ITP Level 1** (Intermediate to Advanced), the
institutional test that Indonesian universities commonly use. Node prefix: `itp-`.

## Format (Level 1)

All questions are multiple choice with four options (A)–(D). The test is available in **paper and digital** format, and the
handbook also describes **at-home testing** under the "Bring Your Own Proctor (BYOP)" procedures run by the local institution.

| Section | Parts / question types | Questions | Time | Scale |
|---|---|---|---|---|
| 1. Listening Comprehension | Part A short conversations (one question each); Part B longer conversations; Part C talks and lectures. Audio is played **once**. | 50 | ~35 min (paced by the recording) | 31–68 |
| 2. Structure and Written Expression | Structure: complete the sentence (Q1–15). Written Expression: choose the one underlined part that must change (Q16–40). | 40 | 25 min | 31–68 |
| 3. Reading Comprehension | Academic passages. Question kinds in ETS samples: main idea/purpose, detail, NOT/EXCEPT, reference, vocabulary in context, inference, location in passage. | 50 | 55 min | 31–67 |
| **Total** | | **140** | **115 min** | **310–677** |

Other rules from the ETS handbook: scores count correct answers only (**no penalty for guessing**). Marking two answers
counts as wrong. There is no scheduled break. Notes are allowed during Listening and Speaking (in the paper test book, or on an
erasable whiteboard at home). Some unscored pretest or equating questions may be included.

**Optional Speaking test** (digital only, sold as an add-on to Level 1 or 2, about 15 min, scale 31–68, AI-scored). It has
four tasks: a read-aloud announcement (60 s preparation, 60 s response), two familiar-topic questions (30 s preparation, 45 s
response each), and one integrated conversation summary (45 s preparation, 60 s response). It is reported separately, and the
handbook says no certificate is available for it yet. The track covers it in a single overview node (`itp-speaking-optional`),
not a full level, because most Indonesian ITP administrations are the three-section test.

**Level 2** (High Beginner to Intermediate: 95 questions, 70 min, 200–500) is **not** covered by this track.

## Scoring and official CEFR mapping

Raw correct answers are converted to scaled section scores. Total = (sum of the three section scores) × 10 ÷ 3. This is the
method in the ETS Practice Test book, e.g. (48+56+57) × 10/3 ≈ 537. Scores are valid for **2 years**. ETS sets **no pass mark**,
and each institution sets its own requirement.

ETS minimum scores per CEFR level (Level 1):

| CEFR | Total | Listening | Structure & WE | Reading | Speaking |
|---|---|---|---|---|---|
| C1 | 620 | 62 | 64 | 60 | 64 |
| B2 | 543 | 55 | 53 | 55 | 58 |
| B1 | 433 | 46 | 43 | 41 | 48 |
| A2 | 343 | 38 | 32 | 33 | 41 |

## Who takes it in Indonesia

University students and staff take it for graduation requirements, placement, scholarships and some job and civil-service
selections. Each institution sets its own threshold. I found no national standard, and the targets of roughly 450–550 often quoted by prep sites are unverified. **IIEF** (Indonesian International
Education Foundation) is ETS's representative for TOEFL ITP in Indonesia. It runs a network of authorised test centres at
universities and language institutions, offering paper-based, digital and online remote-proctored sessions. ITC (International
Test Center) represents ETS for other tests (TOEIC, TOEFL iBT/Junior/Primary), not ITP. ETS says ITP is **not** a substitute for
TOEFL iBT, and its scores are reported only to the administering institution.

## How the track maps to the test

`recommendedAfter`: English **B1**. Level 1 is "intermediate to advanced", and the B1 cut (433) is the lowest band most
graduation targets sit above.

| Track level | Covers |
|---|---|
| `Start` | Format, scoring/CEFR, guessing, elimination and pacing |
| `Listening` | Part A, idioms and implied meaning, Part B, Part C, note-taking and prediction |
| `Structure` | Structure (clauses, verb forms, inversion/conditionals/comparisons), Written Expression (error spotting, agreement/parallelism), section pacing |
| `Reading` | Main idea, detail/EXCEPT/location, vocabulary in context, reference, inference, academic vocabulary, pacing |
| `Mock` | Optional Speaking overview, Mock Test 1, score review and error log, Mock Test 2 and test-day plan |

25 nodes in total. Practice scores in the app are **estimates**. Official conversion tables vary by test form, so FluentPath must
never present an estimate as an ETS score.

## Uncertain or unverified

- **Listening part split.** ETS web pages give only the total of 50 questions. In ETS's *Official TOEFL ITP Practice Tests,
  Level 1, Vol. 3* (2020), Part A is Q1–30, Part B is Q31–37 (2 conversations) and Part C is Q38–50 (3 talks). Many prep
  sites quote 30/8/12 instead, so the B/C split may vary by form. The track does not hard-code it.
- **Number of reading passages.** ETS does not state it. The Vol. 3 practice tests use 5 passages of 8–11 questions each.
- **Listening timing.** It is "approximately 35 minutes" and paced by the recording, not a fixed timer.
- **Underlined vs highlighted.** The paper test underlines the four Written Expression parts, while the digital sample
  questions "highlight" them.
- **Indonesian delivery.** IIEF's own site (iief.or.id) returned HTTP 403, so the details of its test-centre network and online
  remote sessions come from Indonesian university pages and news, not from IIEF directly.
- **No format change found.** The ETS ITP pages (checked 2026-10-15) and the © 2025 handbook show the same Level 1 format, and I
  found no ETS notice of an ITP change. The widely reported January 2026 redesign is of TOEFL iBT, a different test, according to
  prep-site coverage; the ETS iBT page I fetched showed only a banner, so this is not confirmed from ETS.

## Sources (accessed 2026-10-15)

- ETS, TOEFL ITP Test Content: https://www.ets.org/toefl/itp/test-content.html
- ETS, TOEFL ITP Scoring (CEFR cut scores, 2-year validity): https://www.ets.org/toefl/itp/scoring.html
- ETS, About TOEFL ITP (uses, not a replacement for iBT): https://www.ets.org/toefl/itp/about.html
- ETS, TOEFL ITP Preparation: https://www.ets.org/toefl/itp/prepare.html
- ETS, TOEFL ITP Test Taker Handbook (© 2025): https://www.de.ets.org/content/dam/ets-org/pdfs/toefl-itp-test-taker-handbook.pdf
- ETS, Official TOEFL ITP Practice Tests Level 1 Vol. 3 ebook (© 2020; part numbering, score calculation): https://www.ets.org/s/toefl-itp/l1v3_ebook/itp-practice-test-level-1-volume-3-ebook.pdf
- ETS sample questions, Level 1 Listening / Structure & Written Expression / Reading (redirected to toefl-samples.ets-rschtech-prod.c.ets.org):
  - https://rt.ets.org/toefl_www/toefl_itp/test_preparation/sample_questions/level1_section1_listening_comprehension.html
  - https://rt.ets.org/toefl_www/toefl_itp/test_preparation/sample_questions/level1_section2_structure_written_expression.html
  - https://rt.ets.org/toefl_www/toefl_itp/test_preparation/sample_questions/level1_section3_reading_comprehension.html
- ETS, TOEFL ITP Speaking Sample Test: https://rt.ets.org/toeflsvr/tpServerRoot/wb/itpSpeak/practice/APWwebPage.html
- Indonesian context (secondary): https://fib.undip.ac.id/?p=254105 (UNDIP and IIEF online ITP, July 2026); https://lac.telkomuniversity.ac.id/events/toefl-itp-online-remote-proctoring-11 (remote-proctored ITP, 2026); https://www.liputan6.com/regional/read/5513878/cara-mudah-kenali-penyelenggara-tes-toefl-resmi (IIEF authorised test centres)
