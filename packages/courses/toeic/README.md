# TOEIC Prep track (`toeic`): research notes

Roadmap only. Every lesson is `coming_soon`. Facts below were checked on 15 October 2026 against the ETS documents listed under Sources.

## What TOEIC is
TOEIC is ETS's English test for the workplace and everyday life. The handbook says it needs no vocabulary beyond everyday workplace use. The TOEIC program has three parts:
- **TOEIC Listening and Reading (L&R).** The main test, and the main part of this track.
- **TOEIC Speaking and Writing (S&W).** Two separate tests with their own scores, covered in the later levels of this track.
- **TOEIC Bridge.** A test for lower levels. This track does not cover it.

## TOEIC Listening and Reading format
200 multiple-choice questions in about 2 hours, with four options (A–D), except Part 2, which has three (A–C). There is no penalty for guessing. The audio is played once and answers go on a separate answer sheet. In the paper test, note paper is not allowed.

| Section | Part (official name) | Questions | Notes |
|---|---|---|---|
| Listening (~45 min, 100 q) | 1 Photographs | 6 | 4 spoken statements per photo |
| | 2 Question-Response | 25 | 3 spoken responses |
| | 3 Conversations | 39 | 13 conversations × 3 questions; some 3-speaker conversations and "Look at the graphic" questions |
| | 4 Talks | 30 | 10 talks × 3 questions |
| Reading (75 min, 100 q) | 5 Incomplete Sentences | 30 | |
| | 6 Text Completion | 16 | 4 texts × 4 gaps; one gap per text needs a whole sentence (handbook sample) |
| | 7 Reading Comprehension: Single Passages | 29 | 10 texts, 2–4 questions each (includes text-message and online-chat texts) |
| | 7 Reading Comprehension: Multiple Passages | 25 | 5 sets of double or triple passages × 5 questions |

**Question types in the official samples** that the track teaches separately:
- "What does the speaker/writer mean when he says/writes…" (implied meaning) in Listening and in Part 7.
- "closest in meaning to" (vocabulary) in Part 7.

## TOEIC Speaking and Writing format
Both tests are taken on a computer. Each is scored 0–200 in steps of 10, and no combined score is reported.

| Test | Questions | Task (official name) | Prep / response time |
|---|---|---|---|
| Speaking (~20 min, 11 q) | 1–2 | Read a text aloud | 45 s / 45 s |
| | 3–4 | Describe a picture | 45 s / 30 s |
| | 5–7 | Respond to questions | 3 s / 15, 15, 30 s |
| | 8–10 | Respond to questions using information provided | 45 s reading; 3 s / 15, 15, 30 s (Q10 heard twice) |
| | 11 | Express an opinion | 45 s / 60 s |
| Writing (~60 min, 8 q) | 1–5 | Write a sentence based on a picture | 8 min for all five; must use the two given words |
| | 6–7 | Respond to a written request | 10 min each (e-mail) |
| | 8 | Write an opinion essay | 30 min; "typically" at least 300 words |

**Rating scales:**
- Speaking: Questions 1–10 are rated 0–3 and Question 11 is rated 0–5.
- Writing: Questions 1–5 are rated 0–3, Questions 6–7 are rated 0–4 and Question 8 is rated 0–5.
- The ratings are converted to the 0–200 scale.

## Scoring and official CEFR mapping
L&R gives a scaled score of 5–495 per section, in steps of 5, and a total of 10–990. Scores are equated per test form. ETS does not publish raw-to-scaled conversion tables, so **practice-test scores can only be estimates**. The score report also gives a percent correct for each "ability measured", such as "details in short spoken texts" or "inferences based on written texts". Node `toeic-lr-score-review` uses these abilities.

The table below gives the ETS minimum score for each CEFR level ("Mapping the TOEIC tests on the CEFR", ETS, © 2025, based on a standard-setting study by Tannenbaum & Wylie):

| Section | Scale | A1 | A2 | B1 | B2 | C1 |
|---|---|---|---|---|---|---|
| Listening | 5–495 | 60 | 110 | 275 | 400 | 490 |
| Reading | 5–495 | 60 | 115 | 275 | 385 | 455* |
| Speaking | 0–200 | 50 | 90 | 120 | 160 | 180 |
| Writing | 0–200 | 30 | 70 | 120 | 150 | 180 |

\* ETS footnotes:
- The Reading C1 cut is based on 45% of the panelists.
- Some Speaking and Writing values "were adjusted from the recommended study values". The extracted PDF text does not show exactly which cells this applies to.

ETS gives no C2 cut. It says the cuts are guidelines rather than rigid cut scores, and that CEFR levels should be read per section. ETS does not publish total-score bands for each CEFR level in this document. Totals such as "B1 ≈ 550" are only the sum of the section cuts and are not an official figure.

## Who takes it (Indonesia)
- **ITC (International Test Center)** runs TOEIC in Indonesia for ETS. ITC describes itself as ETS's official representative and offers TOEIC L&R, TOEIC S&W and TOEIC Bridge. It lists L&R as "paper-and-pencil or computer based", and its homepage lists the L&R Public Testing Program at Rp675.000 (seen 15 Oct 2026; prices change).
- **Who uses the scores:**
  - Employers use them for recruitment, promotion and overseas assignments. ITC's testimonials cite the Indonesia Stock Exchange, PT Pertagas Niaga and PT GMF AeroAsia.
  - The tourism and hospitality sector uses them, including the Ministry of Tourism.
  - Universities and polytechnics use them as exit or graduation requirements; Politeknik Negeri Bandung is a test centre.
  - Vocational schools (SMK) use them.
- Assumption, not verified with data: most Indonesian requirements ask for **L&R only**, which is why S&W comes last in this track.

## How the track's levels map to the test
| Level id | Covers | Nodes |
|---|---|---|
| `Start` | Format, workplace vocabulary, CEFR-based goal | 3 |
| `Listening` | Parts 1–4 + paraphrase traps + graphic/implied questions | 6 |
| `Reading` | Word forms + Parts 5, 6, 7 (single, multiple) + time management | 6 |
| `Mock` | Full L&R practice test, error review, test-day plan | 3 |
| `Speaking` | One node per official Speaking task (Q1–11) | 5 |
| `Writing` | One node per official Writing task (Q1–8) + S&W practice test | 4 |

That is 27 nodes in total.
- `recommendedAfter` is English **B1**. The TOEIC L&R B1 cut (275/275) is roughly where the test content (workplace e-mails, talks, double/triple passages) becomes workable.
- The Speaking and Writing levels only require `toeic-format-overview`, because S&W is a separate test that some learners take on its own.

## Uncertain / not verified
- **Delivery:**
  - The ETS L&R handbook used here is the *paper-delivered* edition, and the S&W handbook is the *computer-delivered* edition.
  - The ETS FAQ only says TOEIC "online formats" exist and that availability varies by region. Remote testing is mentioned in the S&W handbook ("contact your local EPN member").
  - ITC also promotes "TOEIC Link", a newer ETS product. I did not verify whether it is a different test, so this track does not cover it.
  - What is available in Indonesia should be confirmed with ITC.
- **Accents:** Third-party sources (EF, ETS Global blog) say Listening uses American, British, Canadian and Australian accents. I did not find this in the ETS handbook, so the objective only says "different English accents".
- **Format changes:** No 2025–2026 change to the TOEIC L&R or S&W format was found. The 2026 changes in the news concern TOEFL iBT, not TOEIC. The current L&R part structure is what ITC calls "the Updated Version of the TOEIC Listening and Reading Test". I did not verify the exact year each country introduced it.
- **Time budgets:** The per-part time budgets in `toeic-reading-time-management` (≈10/8/55 min) are common prep advice, not ETS guidance.

## Sources (accessed 15 October 2026)
- ETS, TOEIC Listening & Reading test page: https://www.ets.org/toeic/test-takers/about/listening-reading.html
- ETS, *Examinee Handbook: TOEIC Listening & Reading Tests, Paper Delivered* (© 2025), with format table, sample questions, scoring, abilities measured and test-day rules: https://www.ets.org/content/dam/ets-org/pdfs/toeic/toeic-listening-reading-test-examinee-handbook.pdf
- ETS, TOEIC Speaking & Writing test page: https://www.ets.org/toeic/test-takers/about/speaking-writing.html
- ETS, *Examinee Handbook: TOEIC Speaking & Writing Tests, Computer Delivered*, with tasks, timings, criteria and 0–200 scale: https://www.ets.org/content/dam/ets-org/pdfs/toeic/toeic-speaking-writing-examinee-handbook.pdf
- ETS, *TOEIC Speaking & Writing Sample Tests*: https://www.ets.org/content/dam/ets-org/pdfs/toeic/toeic-speaking-writing-sample-tests.pdf
- ETS, *Mapping the TOEIC tests on the CEFR* (© 2025): https://www.ets.org/pdfs/toeic/toeic-mapping-cefr-reference.pdf
- ETS, score use resources (links to the mapping and score guides): https://www.eu.ets.org/toeic/resources/scores/score-use.html
- ETS, TOEIC test-taker FAQ (online formats vary by region): https://www.ets.org/toeic/about/faq/test-taker-related-faq.html
- IIBC (TOEIC program in Japan), L&R format page, which confirms the part counts: https://www.iibc-global.org/english/toeic/test/lr/about/format.html
- ITC Indonesia, TOEIC page and homepage: https://www.itc-indonesia.com/toeic/ , https://www.itc-indonesia.com/
- Third-party, for accents only: EF TOEIC listening guide, https://www.ef.com/wwen/english-tests/toeic/listening/
