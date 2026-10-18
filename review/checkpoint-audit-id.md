# Audit checkpoint FluentPath — 16 Oktober 2026

Branch: `ui-dialog-colour-icons`. Cakupan: melanjutkan tahap 0–6 dari plan Claude; akar masalahnya bentuk jawaban yang tidak cocok untuk pemula, format yang dihukum sebagai kesalahan isi, dan materi yang belum diajarkan. Belum digabung atau diterapkan ke produksi.

## Hasil dan keputusan

- Jenis baru tetap khusus checkpoint: `letters`, `multi`, `match`. Langkah pelajaran tetap `mcq` / `word_order`.
- A0: tanpa huruf pengecoh, maksimal satu jawaban ketik per checkpoint pelajaran (dua untuk checkpoint level), setiap jawaban ketik maksimal tiga kata. A1: maksimal dua pengecoh dan dua soal ketik per pelajaran (tiga untuk level).
- 263 item A0–A1 yang sebelumnya teks bebas dialihkan ke interaksi ketuk. Jumlah item dan cakupan objective tiap pelajaran tetap. A0 sekarang 7/84 item teks; A1 11/202. Item tulisan A1 yang dipertahankan memang melatih produksi bahasa dengan pola yang sudah diajarkan.
- A2–C2 tidak dirombak tingkat kesulitannya. Hanya petunjuk format yang dibetulkan; format tanggal dan house style C2 tetap diuji.
- `a dan e`, `A, E`, dan `E and A` pada soal daftar huruf lama mendapat nilai penuh. Huruf tambahan, huruf kurang, atau urutan ejaan salah tidak otomatis diterima.
- Penilai AI membedakan format sampingan dengan format yang menjadi target. Normalisasi server tidak boleh memberi nilai penuh otomatis pada soal house style; penilai cadangan juga menjaga tanda baca dan kapitalisasi yang diuji.

## Tahapan yang diteruskan

Tahap 0–2 sudah memiliki commit di branch: `64ab7d3` (objective tidak membocorkan jawaban), `599981a` dan `48a0da2` (soal satu tugas, level, please/room), `fcc6663` (server), `8e1c34f` (UI), `d3f2a32` (label panjang). Commit `72e2f3b` menyimpan pekerjaan awal berikutnya. Pekerjaan itu dipertahankan dan dilengkapi, bukan dibuang.

Tahap 3: writer/verifier memiliki aturan bentuk, batas teks dan pengecoh yang sama dengan validator. Jawaban ketik A0 yang tidak ada dalam materi siswa ditolak; satu nama huruf alfabet tetap sah. Catatan guru bukan bukti bahwa siswa sudah mempelajari suatu istilah. Retry menerima masalah dari draft terbaru, sambil mempertahankan alasan spesifik pemeriksa.

Tahap 4: audit semua lesson, template latihan, soal material, dan langkah EN/ID; perbaikan authored assessment, panduan penulis, dan validasi saat coursecheck/startup. Rincian di bawah.

Tahap 5: instruksi penilaian isi/format diperjelas, kompatibilitas daftar huruf lama ditambahkan, dan pengecualian format C2 diuji.

Tahap 6: driver e2e menjawab keenam tipe dengan payload sah, tes database/server ditambah, browser diperiksa, dan OpenAI asli diuji terbatas tanpa TTS. Checkbox/radio serta baris pasangan kini menyampaikan status pilihan di web.

## Perubahan isi yang penting

| Masalah | Perbaikan |
| --- | --- |
| `alpha-4`, `alpha-5` meminta strip / format huruf | Dengarkan ejaan, ketuk huruf; dua E pada green tetap dua tile. Pengecoh A0 dihapus. Catatan guru dan pengantar tidak lagi meminta strip. |
| `alpha-3`: beberapa huruf vokal dijawab sebagai teks | Multi: pilih A dan E dari huruf table. |
| `wb-1`, `wa-4`: daftar kata yang memerlukan kapital | Multi dengan seluruh kata yang benar sebagai kunci. Soal lain memilih satu kata / tanda baca yang jelas. |
| `vocab-2` | Match table/book/phone dengan meja/buku/telepon. |
| `rst-6` | Multi untuk seluruh hari kelas renang pada pemberitahuan. |
| Urutan kartu pos `wa-7` | Pilih satu urutan bagian yang benar; tidak perlu mengetik format B/D/C/A. |
| A0 meminta kalimat buatan sendiri | Pilihan dari bentuk yang sudah diajarkan; susun kata hanya untuk kalimat pendek yang jelas. |
| Please diperlakukan sebagai aturan posisi universal; water belum dijelaskan | Jelaskan dua pola yang sedang dilatih dan arti water sebelum pertanyaan; penjelasan tidak menyatakan posisi lain selalu salah. |
| Pronoun kepemilikan ambigu | Mrs Lee secara eksplisit pemilik kebun. |
| And/but/because atau any/some sama-sama mungkin | Instruksi menjelaskan makna kontras/alasan, atau pengecoh diganti dengan bentuk yang jelas salah. Negatif dengan and tidak disebut selalu tidak gramatikal. |
| Guitar/novels/rice sebagai target belum diajarkan | Gunakan cooking atau contoh bus yang sudah diajarkan. |
| Referensi terbuka terlalu berupa awalan: `i watch`, `in the evening i` | Referensi kalimat lengkap; jawaban lain tetap dinilai berdasarkan rubrik. |
| Apostrof hilang dalam template contraction/reading | Referensi diperbaiki; contraction yang memang diuji tidak mengajarkan `isnt`/`doesnt`. |
| Sapaan email seakan melarang koma | B1 emails, B2 formal/informal vocab dan template B2 formal writing menyatakan koma opsional. |
| C2 figurative template mewajibkan tanda hubung sampingan | Petunjuk meminta ungkapan yang benar; referensi light-fingered/light fingered tetap diterima. |

## Inventaris A0–A1 setelah perbaikan

MCQ = pilih satu; urut = susun kata; huruf = eja; multi = pilih semua; match = pasangkan.

| Pelajaran | MCQ | Urut | Huruf | Multi | Match | Teks |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `a0-greetings` | 5 | 0 | 0 | 0 | 0 | 1 |
| `a0-alphabet-sounds` | 2 | 0 | 2 | 1 | 0 | 0 |
| `a0-sounds-indonesian` | 7 | 0 | 0 | 0 | 0 | 0 |
| `a0-classroom-language` | 5 | 0 | 0 | 0 | 0 | 1 |
| `a0-pronouns` | 4 | 0 | 0 | 0 | 0 | 1 |
| `a0-common-vocabulary` | 3 | 0 | 0 | 0 | 1 | 1 |
| `a0-numbers` | 6 | 0 | 0 | 0 | 0 | 0 |
| `a0-time-dates` | 6 | 0 | 0 | 0 | 0 | 0 |
| `a0-personal-info` | 7 | 0 | 0 | 0 | 0 | 0 |
| `a0-to-be-basics` | 6 | 0 | 0 | 0 | 0 | 0 |
| `a0-basic-sentences` | 4 | 1 | 0 | 0 | 0 | 0 |
| `a0-writing-basics` | 5 | 0 | 0 | 1 | 0 | 1 |
| `a0-listening-first` | 6 | 0 | 0 | 0 | 0 | 1 |
| `a0-reading-first` | 5 | 0 | 0 | 0 | 0 | 1 |
| `a1-to-be` | 5 | 1 | 0 | 0 | 0 | 0 |
| `a1-articles-demonstratives` | 6 | 0 | 0 | 0 | 0 | 0 |
| `a1-adjectives` | 5 | 1 | 0 | 0 | 0 | 0 |
| `a1-possessives` | 6 | 0 | 0 | 0 | 0 | 1 |
| `a1-there-is-are` | 7 | 0 | 0 | 0 | 0 | 0 |
| `a1-simple-present` | 2 | 2 | 0 | 0 | 0 | 1 |
| `a1-have-got` | 7 | 0 | 0 | 0 | 0 | 0 |
| `a1-adverbs-frequency` | 5 | 2 | 0 | 0 | 0 | 0 |
| `a1-prepositions` | 7 | 0 | 0 | 0 | 0 | 0 |
| `a1-questions` | 2 | 3 | 0 | 0 | 0 | 0 |
| `a1-countable-some-any` | 6 | 0 | 0 | 0 | 0 | 0 |
| `a1-present-continuous` | 6 | 0 | 0 | 0 | 0 | 0 |
| `a1-can-ability` | 6 | 1 | 0 | 0 | 0 | 0 |
| `a1-imperatives` | 6 | 0 | 0 | 0 | 0 | 0 |
| `a1-like-would-like` | 5 | 0 | 0 | 0 | 0 | 2 |
| `a1-past-be` | 6 | 1 | 0 | 0 | 0 | 0 |
| `a1-conjunctions` | 7 | 0 | 0 | 0 | 0 | 0 |
| `a1-vocab-family-home` | 7 | 0 | 0 | 0 | 0 | 0 |
| `a1-vocab-food-shopping` | 6 | 0 | 0 | 0 | 0 | 0 |
| `a1-vocab-town-transport` | 7 | 0 | 0 | 0 | 0 | 0 |
| `a1-vocab-jobs-hobbies` | 7 | 0 | 0 | 0 | 0 | 0 |
| `a1-vocab-daily-life` | 6 | 0 | 0 | 0 | 0 | 0 |
| `a1-pron-endings` | 6 | 0 | 0 | 0 | 0 | 1 |
| `a1-pron-word-stress` | 6 | 0 | 0 | 0 | 0 | 0 |
| `a1-listening-everyday` | 7 | 0 | 0 | 0 | 0 | 0 |
| `a1-reading-short-texts` | 6 | 0 | 0 | 1 | 0 | 0 |
| `a1-self-introduction` | 6 | 0 | 0 | 0 | 0 | 1 |
| `a1-daily-routines` | 6 | 0 | 0 | 0 | 0 | 0 |
| `a1-writing-about-me` | 4 | 0 | 0 | 1 | 0 | 2 |
| `a1-everyday-conversation` | 5 | 0 | 0 | 0 | 0 | 2 |
| `a1-mediation-basic` | 5 | 0 | 0 | 0 | 0 | 1 |

## Metode dan batas audit

Jalankan `python3 scripts/checkpoint_audit.py --output review/checkpoint-audit-candidates.json`.

Inventaris akhir: **5,950 entri soal dari 187 lesson**, termasuk 1269 authored assessment, 1357 template latihan, 1734 soal langkah EN/ID, serta 1590 soal material EN/ID. Pemindai menghasilkan **136 kandidat**, bukan 136 kesalahan.

Pencarian kosakata menggunakan material/pola siswa dan prerequisite, serta materi level sebelumnya. Assessment keys dan teaching notes tidak dianggap sebagai materi siswa. Pemindai tidak memahami infleksi, sinonim, nama orang, atau bahasa Indonesia; hasilnya diperiksa dalam konteks. Misalnya house/green boleh dieja dari audio tanpa menguji arti; watches adalah perubahan watch; Bayu/Wahyu/Lestari/Rizky berasal dari ejaan yang diberikan; pm/a.m./p.m. berbeda tokenisasi; kirimu adalah jawaban terjemahan Indonesia. Sinonim benar yang hanya menjadi alternatif referensi tidak berarti siswa wajib menghasilkan kata baru itu.

Template `guidedPractice` / `independentPractice` bukan widget checkpoint dan bukan cara aplikasi menentukan mastery. Soal yang tampil di langkah berasal dari authored plan dan tetap berbentuk ketuk. Template A0 dengan pola tetap dan latihan produksi pendek A1 tetap dapat digunakan sebagai contoh pengajaran; batas ketat checkpoint diterapkan pada assessment dan output AI. Pilihan yang disebut dalam template pengenalan kini ditulis lengkap, bukan instruksi memilih tanpa pilihan.

Daftar berikut mencatat seluruh kandidat yang tersisa beserta keputusan tinjauannya. Kapitalisasi/tanda baca yang memang menjadi tujuan bukan kesalahan format sampingan.

| Lokasi | Temuan pemindai | Keputusan |
| --- | --- | --- |
| `a0-alphabet-sounds` assessment `alpha-4` | house | Ejaan huruf diberikan audio; arti kata bukan yang diuji. |
| `a0-alphabet-sounds` assessment `alpha-5` | green | Ejaan huruf diberikan audio; arti kata bukan yang diuji. |
| `a0-alphabet-sounds` guidedPractice `guidedPractice-3` | format_or_capitalization_target, spelling_in_free_text | Template pengenalan memiliki pilihan eksplisit; strip tidak diwajibkan. Checkpoint memakai tile. |
| `a0-alphabet-sounds` independentPractice `independentPractice-2` | format_or_capitalization_target, spelling_in_free_text | Template pengenalan memiliki pilihan eksplisit; strip tidak diwajibkan. Checkpoint memakai tile. |
| `a0-basic-sentences` assessment `sent-5` | watches | Infleksi/kepemilikan atau bentuk kata yang mengikuti pola; bukan aturan baru. |
| `a0-basic-sentences` guidedPractice `guidedPractice-3` | beginner_production | Template pola tetap/ungkapan pendek; langkah dan checkpoint yang tampil memakai ketuk. |
| `a0-basic-sentences` independentPractice `independentPractice-2` | cats | Infleksi/kepemilikan atau bentuk kata yang mengikuti pola; bukan aturan baru. Template pola tetap/ungkapan pendek; langkah dan checkpoint yang tampil memakai ketuk. |
| `a0-classroom-language` independentPractice `independentPractice-1` | could | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. |
| `a0-common-vocabulary` guidedPractice `guidedPractice-3` | beginner_production | Template pola tetap/ungkapan pendek; langkah dan checkpoint yang tampil memakai ketuk. |
| `a0-common-vocabulary` independentPractice `independentPractice-2` | large | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. Template pola tetap/ungkapan pendek; langkah dan checkpoint yang tampil memakai ketuk. |
| `a0-greetings` independentPractice `independentPractice-1` | goodnight | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. |
| `a0-listening-first` assessment `lis-1` | wahyu | Nama/email diberikan melalui ejaan atau konteks. |
| `a0-listening-first` guidedPractice `guidedPractice-1` | bayu | Nama/email diberikan melalui ejaan atau konteks. |
| `a0-listening-first` guidedPractice `guidedPractice-4` | miss, sir | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. |
| `a0-personal-info` assessment `pi-5` | lestari | Nama/email diberikan melalui ejaan atau konteks. |
| `a0-personal-info` independentPractice `independentPractice-1` | beginner_production | Template pola tetap/ungkapan pendek; langkah dan checkpoint yang tampil memakai ketuk. |
| `a0-personal-info` independentPractice `independentPractice-2` | beginner_production | Template pola tetap/ungkapan pendek; langkah dan checkpoint yang tampil memakai ketuk. |
| `a0-pronouns` guidedPractice `guidedPractice-3` | format_or_capitalization_target | Kapitalisasi/tanda baca adalah objective; pertanyaan pengenalan atau pilihan tersedia. |
| `a0-to-be-basics` guidedPractice `guidedPractice-2` | beginner_production | Template pola tetap/ungkapan pendek; langkah dan checkpoint yang tampil memakai ketuk. |
| `a0-to-be-basics` independentPractice `independentPractice-1` | beginner_production | Template pola tetap/ungkapan pendek; langkah dan checkpoint yang tampil memakai ketuk. |
| `a0-to-be-basics` independentPractice `independentPractice-2` | beginner_production | Template pola tetap/ungkapan pendek; langkah dan checkpoint yang tampil memakai ketuk. |
| `a0-writing-basics` assessment `wb-1` | format_or_capitalization_target | Kapitalisasi/tanda baca adalah objective; pertanyaan pengenalan atau pilihan tersedia. |
| `a0-writing-basics` assessment `wb-2` | format_or_capitalization_target | Kapitalisasi/tanda baca adalah objective; pertanyaan pengenalan atau pilihan tersedia. |
| `a0-writing-basics` guidedPractice `guidedPractice-1` | format_or_capitalization_target | Kapitalisasi/tanda baca adalah objective; pertanyaan pengenalan atau pilihan tersedia. |
| `a0-writing-basics` independentPractice `independentPractice-1` | format_or_capitalization_target | Kapitalisasi/tanda baca adalah objective; pertanyaan pengenalan atau pilihan tersedia. |
| `a0-writing-basics` independentPractice `independentPractice-2` | format_or_capitalization_target | Kapitalisasi/tanda baca adalah objective; pertanyaan pengenalan atau pilihan tersedia. |
| `a0-writing-basics` plan.en `step-2` | format_or_capitalization_target | Kapitalisasi/tanda baca adalah objective; pertanyaan pengenalan atau pilihan tersedia. |
| `a0-writing-basics` plan.id `step-2` | format_or_capitalization_target | Kapitalisasi/tanda baca adalah objective; pertanyaan pengenalan atau pilihan tersedia. |
| `a0-writing-basics` material.en `capitals-block-5` | format_or_capitalization_target | Kapitalisasi/tanda baca adalah objective; pertanyaan pengenalan atau pilihan tersedia. |
| `a0-writing-basics` material.id `capitals-block-5` | format_or_capitalization_target | Kapitalisasi/tanda baca adalah objective; pertanyaan pengenalan atau pilihan tersedia. |
| `a1-adjectives` guidedPractice `guidedPractice-1` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-adjectives` guidedPractice `guidedPractice-4` | bag's | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-adjectives` independentPractice `independentPractice-1` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-adjectives` independentPractice `independentPractice-3` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-adverbs-frequency` guidedPractice `guidedPractice-4` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-adverbs-frequency` independentPractice `independentPractice-1` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-adverbs-frequency` independentPractice `independentPractice-2` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-adverbs-frequency` independentPractice `independentPractice-3` | movies | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. |
| `a1-articles-demonstratives` independentPractice `independentPractice-2` | bicycle, got, i've | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-can-ability` guidedPractice `guidedPractice-4` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-can-ability` independentPractice `independentPractice-1` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-conjunctions` guidedPractice `guidedPractice-4` | got | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. |
| `a1-conjunctions` independentPractice `independentPractice-1` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-conjunctions` independentPractice `independentPractice-2` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-countable-some-any` guidedPractice `guidedPractice-2` | bowls | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-countable-some-any` independentPractice `independentPractice-1` | i've | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-countable-some-any` independentPractice `independentPractice-3` | bowl | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. |
| `a1-daily-routines` independentPractice `independentPractice-2` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-everyday-conversation` independentPractice `independentPractice-1` | could | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. |
| `a1-everyday-conversation` independentPractice `independentPractice-2` | nearby | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. |
| `a1-have-got` guidedPractice `guidedPractice-3` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-have-got` guidedPractice `guidedPractice-4` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-have-got` independentPractice `independentPractice-1` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-imperatives` guidedPractice `guidedPractice-1` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-imperatives` guidedPractice `guidedPractice-2` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-imperatives` guidedPractice `guidedPractice-3` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-imperatives` independentPractice `independentPractice-1` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-imperatives` independentPractice `independentPractice-2` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-like-would-like` assessment `lw-1` | running | Infleksi/kepemilikan atau bentuk kata yang mengikuti pola; bukan aturan baru. |
| `a1-like-would-like` assessment `lw-2` | waiting | Infleksi/kepemilikan atau bentuk kata yang mengikuti pola; bukan aturan baru. |
| `a1-like-would-like` assessment `lw-3` | beginner_production | Produksi A1 pendek sesuai pola; masih di bawah batas teks per lesson. |
| `a1-like-would-like` guidedPractice `guidedPractice-2` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-like-would-like` guidedPractice `guidedPractice-3` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-like-would-like` independentPractice `independentPractice-1` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-listening-everyday` independentPractice `independentPractice-1` | rizky | Nama/email diberikan melalui ejaan atau konteks. |
| `a1-listening-everyday` independentPractice `independentPractice-2` | kirimu | Jawaban terjemahan Indonesia; token Inggris tidak relevan. |
| `a1-mediation-basic` assessment `med-2` | pm | Variasi tokenisasi a.m./p.m. yang sudah diajarkan. |
| `a1-mediation-basic` assessment `med-6` | beginner_production | Produksi A1 pendek sesuai pola; masih di bawah batas teks per lesson. |
| `a1-mediation-basic` guidedPractice `guidedPractice-2` | pm | Variasi tokenisasi a.m./p.m. yang sudah diajarkan. |
| `a1-past-be` guidedPractice `guidedPractice-3` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-past-be` guidedPractice `guidedPractice-4` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-past-be` independentPractice `independentPractice-1` | ill | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-past-be` independentPractice `independentPractice-2` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-possessives` assessment `poss-2` | father's | Infleksi/kepemilikan atau bentuk kata yang mengikuti pola; bukan aturan baru. |
| `a1-possessives` guidedPractice `guidedPractice-1` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-possessives` guidedPractice `guidedPractice-3` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-possessives` independentPractice `independentPractice-1` | brother's, younger | Infleksi/kepemilikan atau bentuk kata yang mengikuti pola; bukan aturan baru. Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-prepositions` guidedPractice `guidedPractice-4` | book's | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-prepositions` independentPractice `independentPractice-1` | chemist, chemist's, drugstore, pharmacy's | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-present-continuous` assessment `pc-3` | dancing | Infleksi/kepemilikan atau bentuk kata yang mengikuti pola; bukan aturan baru. |
| `a1-present-continuous` assessment `pc-6` | wearing | Infleksi/kepemilikan atau bentuk kata yang mengikuti pola; bukan aturan baru. |
| `a1-present-continuous` guidedPractice `guidedPractice-3` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-present-continuous` guidedPractice `guidedPractice-5` | singing | Infleksi/kepemilikan atau bentuk kata yang mengikuti pola; bukan aturan baru. |
| `a1-present-continuous` independentPractice `independentPractice-1` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-present-continuous` independentPractice `independentPractice-3` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-pron-endings` assessment `pe-1` | teaches | Infleksi/kepemilikan atau bentuk kata yang mengikuti pola; bukan aturan baru. |
| `a1-self-introduction` independentPractice `independentPractice-1` | name's | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. |
| `a1-simple-present` assessment `sp-5` | beginner_production | Produksi A1 pendek sesuai pola; masih di bawah batas teks per lesson. |
| `a1-simple-present` guidedPractice `guidedPractice-2` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-simple-present` guidedPractice `guidedPractice-3` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-simple-present` independentPractice `independentPractice-1` | wake | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-simple-present` independentPractice `independentPractice-2` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-there-is-are` guidedPractice `guidedPractice-2` | bedrooms | Infleksi/kepemilikan atau bentuk kata yang mengikuti pola; bukan aturan baru. |
| `a1-there-is-are` guidedPractice `guidedPractice-3` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-there-is-are` independentPractice `independentPractice-1` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-there-is-are` independentPractice `independentPractice-3` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-to-be` guidedPractice `guidedPractice-2` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-to-be` guidedPractice `guidedPractice-3` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-to-be` independentPractice `independentPractice-1` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-vocab-daily-life` independentPractice `independentPractice-1` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-vocab-daily-life` independentPractice `independentPractice-2` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-vocab-daily-life` independentPractice `independentPractice-3` | hurting | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. |
| `a1-vocab-family-home` guidedPractice `guidedPractice-1` | granny | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. |
| `a1-vocab-family-home` guidedPractice `guidedPractice-4` | refrigerator | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. |
| `a1-vocab-family-home` independentPractice `independentPractice-1` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-vocab-food-shopping` guidedPractice `guidedPractice-4` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-vocab-food-shopping` independentPractice `independentPractice-1` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-vocab-food-shopping` independentPractice `independentPractice-3` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-vocab-jobs-hobbies` independentPractice `independentPractice-2` | dancing, singing, watching | Infleksi/kepemilikan atau bentuk kata yang mengikuti pola; bukan aturan baru. |
| `a1-vocab-jobs-hobbies` independentPractice `independentPractice-3` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-vocab-town-transport` independentPractice `independentPractice-1` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-writing-about-me` assessment `wa-1` | beginner_production | Produksi A1 pendek sesuai pola; masih di bawah batas teks per lesson. |
| `a1-writing-about-me` assessment `wa-2` | cooks | Infleksi/kepemilikan atau bentuk kata yang mengikuti pola; bukan aturan baru. |
| `a1-writing-about-me` assessment `wa-3` | got | Alternatif sinonim/contraction yang sah; bukan satu-satunya jawaban yang diwajibkan. Produksi A1 pendek sesuai pola; masih di bawah batas teks per lesson. |
| `a1-writing-about-me` assessment `wa-4` | format_or_capitalization_target | Kapitalisasi/tanda baca adalah objective; pertanyaan pengenalan atau pilihan tersedia. |
| `a1-writing-about-me` guidedPractice `guidedPractice-1` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-writing-about-me` guidedPractice `guidedPractice-2` | format_or_capitalization_target | Kapitalisasi/tanda baca adalah objective; pertanyaan pengenalan atau pilihan tersedia. |
| `a1-writing-about-me` independentPractice `independentPractice-1` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-writing-about-me` independentPractice `independentPractice-2` | format_or_capitalization_target | Kapitalisasi/tanda baca adalah objective; pertanyaan pengenalan atau pilihan tersedia. |
| `a1-writing-about-me` independentPractice `independentPractice-3` | beginner_production | Template latihan pendek pola A1 yang diajarkan; bukan soal terbuka tanpa konteks. |
| `a1-writing-about-me` plan.en `step-5` | format_or_capitalization_target | Kapitalisasi/tanda baca adalah objective; pertanyaan pengenalan atau pilihan tersedia. |
| `a1-writing-about-me` plan.id `step-5` | format_or_capitalization_target | Kapitalisasi/tanda baca adalah objective; pertanyaan pengenalan atau pilihan tersedia. |
| `a1-writing-about-me` material.en `capitals-block-4` | format_or_capitalization_target | Kapitalisasi/tanda baca adalah objective; pertanyaan pengenalan atau pilihan tersedia. |
| `a1-writing-about-me` material.id `capitals-block-4` | format_or_capitalization_target | Kapitalisasi/tanda baca adalah objective; pertanyaan pengenalan atau pilihan tersedia. |
| `a2-zero-first-conditional` plan.en `step-7` | format_or_capitalization_target | Tanda koma memang materi klausa if; soal pengenalan dipertahankan. |
| `a2-zero-first-conditional` plan.id `step-7` | format_or_capitalization_target | Tanda koma memang materi klausa if; soal pengenalan dipertahankan. |
| `a2-zero-first-conditional` material.en `comma-order-block-3` | format_or_capitalization_target | Tanda koma memang materi klausa if; soal pengenalan dipertahankan. |
| `a2-zero-first-conditional` material.id `comma-order-block-3` | format_or_capitalization_target | Tanda koma memang materi klausa if; soal pengenalan dipertahankan. |
| `b1-emails` assessment `em-1` | format_or_capitalization_target | Sapaan diuji; koma sudah dinyatakan opsional. |
| `b2-formal-informal-vocab` assessment `fiv-6` | format_or_capitalization_target | Sapaan diuji; koma sudah dinyatakan opsional. |
| `b2-formal-writing` guidedPractice `guidedPractice-3` | format_or_capitalization_target | Sapaan diuji; koma sudah dinyatakan opsional. |
| `b2-pron-intonation-attitude` independentPractice `independentPractice-1` | format_or_capitalization_target | Kapital menandai kata yang ditekankan; bagian tugas pronunciation. |
| `c2-editing` assessment `ed-3` | format_or_capitalization_target | House style memang target; penilai wajib mempertahankan format. |
| `c2-editing` guidedPractice `guidedPractice-4` | format_or_capitalization_target | House style memang target; penilai wajib mempertahankan format. |
| `c2-editing` material.en `style-guide-block-3` | format_or_capitalization_target | House style memang target; penilai wajib mempertahankan format. |
| `c2-editing` material.id `style-guide-block-3` | format_or_capitalization_target | House style memang target; penilai wajib mempertahankan format. |

## Kompatibilitas, progress, dan audio

- Tidak ada perubahan skema database. Checkpoint yang sudah tersimpan tetap memakai item/kunci versinya sendiri. Attempt lama dan chat tidak dihapus. Jawaban daftar huruf lama menerima variasi separator yang benar.
- Perubahan assessment hanya berlaku pada checkpoint baru. Wording pada plan greetings diperbarui tanpa mengganti jumlah/kind/options/key/listen langkah; progress dipertahankan. Jika pertanyaan/struktur langkah kelak berubah, mekanisme versi yang sudah ada tetap memulai sesi baru sambil menyimpan sesi lama.
- Tidak ada audio dibuat. Alphabet sudah tercantum di `audio/pending.json` dari perubahan sebelumnya; audio baru tetap menunggu review pemilik. Uji browser tidak menekan tombol audio.
- Checkpoint tidak lagi memaksa warming TTS: `WARM_SPEECH` dan `LIVE_SPEECH` dihormati. Uji regresi memeriksa kedua nilai warming saat live speech mati.

## Verifikasi

| Pemeriksaan | Hasil |
| --- | --- |
| Semua tes Go dengan PostgreSQL sementara | Lulus, termasuk validasi/payload/nilai sebagian/hasil/practice, kepemilikan data, chat, progress, dan regresi baru. |
| `go run ./cmd/coursecheck` | 194/194 node English ready valid; empat track coming-soon valid. |
| E2E HTTP mock | 187 lesson + 7 checkpoint level lulus; setiap checkpoint level 100/100. Progress, flashcard, note bertahan setelah restart; akun kedua terisolasi. |
| Typecheck, Expo lint, export web | Lulus. |
| Browser Chrome 1440×1000 EN dan 390×844 ID | 30 soal / enam tipe; ketuk, unselect, huruf ganda, remove, pasangan, kembali dengan draft tersimpan, submit dan hasil 100/100. Card tidak melebar keluar viewport; label CTA panjang membungkus. |
| Audit yang dapat dijalankan ulang | 5.950 entri, 136 kandidat; keputusan di tabel di atas. |

### OpenAI asli: hasil dan penggunaan

Model mengikuti `.env`: `gpt-5.4-mini`. Semua panggilan melalui API/server + PostgreSQL sementara. Service mematikan warm/live speech dan prefetch; transport menolak endpoint selain Responses. **Nol panggilan TTS dan nol percobaan ke endpoint audio**. Key tidak dicetak atau disimpan dalam laporan.

| Uji | Hasil | Panggilan | Input tidak cached + output |
| --- | --- | ---: | ---: |
| Awal | Dua draft alphabet ditolak karena bentuk teks/terlalu panjang; authored fallback aman. | 2 | 5.920 |
| Setelah instruksi diperjelas | Alphabet dan greetings dibuat serta diverifikasi OpenAI. Alphabet juga dinilai OpenAI. Grading greetings memakai fallback karena kuota uji sudah habis; ini tidak dihitung sebagai bukti live grading greetings. Tinjauan manual menemukan frasa untaught dan IPA, lalu diperbaiki. | 6 | 20.126 |
| Verifikasi akhir, generation-only | Draft awal tile kehilangan huruf ganda ditolak; draft pengganti dan verifier lulus. Alphabet akhir: 2 MCQ, 1 multi, 2 letters, 1 match; nol text, nol decoy, tanpa IPA. | 3 | 7.743 |
| **Total seluruh uji** | Tidak ada panggilan live lagi setelah ini. | **11** | **33.789** |

Batas uji: maksimum enam request per run dan berhenti sebelum request berikutnya setelah ambang 20.000 token. Satu request yang sedang berjalan bisa melewati ambang sedikit; pemakaian aktual dicatat di atas. Ini sampel nyata, bukan bukti bahwa semua variasi model selalu benar: validator, verifier, dan authored fallback tetap wajib. Test live bersifat opt-in; tes biasa tidak menyentuh token pemilik. Mode generation-only dapat dipilih dengan `LIVE_CHECKPOINT_GENERATION_ONLY=1`.

Merge/publish dan pembuatan audio tetap menunggu review pemilik sesuai cakupan plan.

## Tindak lanjut dari pengujian pemilik: konteks soal dan kapitalisasi

Pemilik menemukan soal OpenAI “Tuliskan 1 huruf yang benar. Jawab dalam bahasa Inggris.” tanpa petunjuk atau audio, dengan kunci tersembunyi A. Jawaban `a` dinilai salah karena pencocokan awal server masih mempertahankan kapitalisasi, lalu penilai mengarang syarat huruf besar.

- Pencocokan isi sekarang mengabaikan kapitalisasi dan tanda baca sampingan. Prompt yang secara eksplisit menguji uppercase/lowercase, tanda baca, atau house style tetap mengikuti rubrik formatnya.
- Instruksi jawaban huruf tanpa konteks seperti contoh tersebut ditolak validator. Writer/verifier wajib menggunakan petunjuk yang dilihat/didengar siswa; kunci tersembunyi bukan petunjuk.
- Pemeriksa AI harus mengembalikan tepat satu verdict untuk setiap ID soal. Kegagalan provider, kuota, JSON rusak, ID hilang/duplikat/tidak dikenal, atau verdict kosong langsung memakai authored fallback.
- Nilai pasti dari server juga mengendalikan penjelasan dan feedback ketika AI menyanggahnya. Klaim `keyWrong` yang hanya mengubah kapitalisasi tidak membatalkan jawaban benar.
- Tes regresi memutar ulang contoh pemilik dan keluaran penilai yang salah, termasuk feedback kapitalisasi yang dikarang. Semua tes Go/database, coursecheck, dan e2e 194 node lulus; tidak ada panggilan OpenAI/TTS untuk pengujian tindak lanjut ini.
- Hanya hasil percobaan lokal yang dilaporkan diperbaiki dari 89 menjadi 100; data asli dicadangkan secara privat dan raw evaluator response dipertahankan. Tampilan browser telah diverifikasi menunjukkan `a` sebagai benar. Tidak ada penghapusan atau penilaian ulang massal riwayat pengguna.


## Perbaikan bahasa opsi — 18 Oktober 2026

Opsi berupa instruksi pelafalan atau penjelasan aturan mengikuti bahasa UI. Kata/kalimat bahasa Inggris yang sedang diuji tetap
bahasa Inggris (misalnya `ship` / `sheep`, bentuk kata kerja, dan ejaan `double P`). Aturan lama yang mewajibkan semua opsi dalam
bahasa Inggris diperbaiki pada panduan konten serta prompt penulis/pemeriksa AI, termasuk soal serupa dan checkpoint level.

- Sembilan soal langkah A0–A1 pada tujuh lesson memiliki `displayOptions` Indonesia; termasuk soal Rina mengucapkan `I tink so`.
- Delapan cek materi memiliki `i18n.id.options`; empat item checkpoint pelafalan memiliki `i18n.id.assessmentOptions`.
- Opsi asli dan kunci indeks tetap sama. Label terjemahan ikut diacak bersama opsi; tampilan jawaban, hint, Ask AI, kartu kesalahan,
  contoh generator, verifier, dan grader memakai label yang dilihat siswa.
- Penambahan label merupakan koreksi wording: sesi lama diperbarui di tempat, jawaban/progres tetap, dan pindah EN ↔ ID tetap
  mempertahankan sesi. Validasi menolak label kosong, duplikat, atau jumlah yang tidak cocok.
- Verifikasi: semua tes Go (termasuk database dalam schema terpisah), `coursecheck`, serta e2e 194 node dengan AI mock berhasil.
  Tampilan cek materi Indonesia diperiksa di browser lokal. Tidak ada panggilan OpenAI atau TTS dalam pengujian.

## Audit seluruh opsi A0–C2 — 18 Oktober 2026

Permintaan lanjutan memperluas pemeriksaan ke seluruh 1.743 soal beropsi: 695 soal langkah, 795 cek materi,
serta 253 item checkpoint authored. Setiap prompt dan opsi dibaca dalam konteksnya; klasifikasi tidak ditentukan dari
panjang teks atau bahasa opsi saja. Empat course persiapan tes belum memiliki lesson siap pakai; kebijakan bahasa kelima course
juga diperjelas agar opsi penjelasan/strategi berbahasa Indonesia, sedangkan soal resmi yang melatih bahasa Inggris tetap Inggris.

Ditemukan dan diterjemahkan **232 soal tambahan pada 78 pelajaran**. Digabung dengan perbaikan awal, totalnya **253 soal pada 85
pelajaran**: 115 langkah latihan, 131 cek materi, dan 7 item checkpoint. Jumlah ini menghitung soal, bukan masing-masing tombol opsi.

| Level | Soal beropsi ditinjau | Soal dengan label Indonesia |
| --- | ---: | ---: |
| A0 | 178 | 10 |
| A1 | 412 | 35 |
| A2 | 225 | 34 |
| B1 | 275 | 32 |
| B2 | 270 | 39 |
| C1 | 220 | 43 |
| C2 | 163 | 60 |
| Total | 1.743 | 253 |

Yang diterjemahkan: instruksi posisi lidah/bunyi, ketukan dan tekanan kata, strategi membaca/menyimak, penjelasan aturan,
interpretasi cerita dan nada bicara, arti idiom, kategori bagian teks, serta langkah penyuntingan. Contoh: `Both are correct`
menjadi `Keduanya benar`; `Read the title and the first sentence of each paragraph` menjadi `Baca judul dan kalimat pertama tiap
paragraf`; `She disagrees with halving the budget` menjadi `Dia tidak setuju anggarannya dipotong setengah`. Nama kategori resmi
seperti `Grammatical Range and Accuracy` tetap dicantumkan bersama arti Indonesianya.

Yang dipertahankan: bentuk grammar yang dibandingkan, kata yang harus dipilih untuk melengkapi kalimat Inggris, contoh respons
untuk percakapan Inggris, kutipan sumber yang harus dikenali, ejaan/nama huruf, target kosakata Inggris, dan data seperti nama,
angka, jam, simbol atau opsi yang sebelumnya sudah Indonesia. Tidak menerjemahkan perbedaan `can` / `can't`, `had` / `would`, atau
bentuk benar/salah tata bahasa menjadi padanan Indonesia yang akan menghilangkan hal yang diuji.

Semua plan EN, opsi asli, indeks kunci, item checkpoint asli, dan teks audio sama dengan HEAD. Label tampilan tetap terpisah,
sehingga koreksi wording mempertahankan progres sesi. Tombol opsi materi, MCQ latihan/checkpoint, dan multi kini dibatasi lebar
panel; teks panjang membungkus tanpa terpotong, diperiksa pada viewport 390 piksel. Tidak ada perubahan audio.

Verifikasi:

- Tes Go baru membaca seluruh label di semua level, memeriksa kesamaan struktur EN/ID, kompatibilitas sesi lama, pengacakan,
  kunci, dan nilai checkpoint. Seluruh tes Go dijalankan ulang dengan `-count=1`, termasuk database dalam schema terpisah.
- `coursecheck` lolos untuk semua course; 194 node English siap.
- E2E lolos untuk seluruh 194 node secara terpisah dalam mode EN dan ID. Run ID memeriksa seluruh 253 soal berlabel terjemahan
  melalui HTTP, termasuk materi, latihan, checkpoint gagal/lulus, progres setelah restart, dan isolasi pengguna.
- Typecheck, Expo lint, dan export web lolos; tampilan opsi panjang Indonesia diperiksa pada browser lokal.
- Generator/verifier dan kebijakan bahasa diperbarui, tetapi pengujian tidak memanggil OpenAI asli atau TTS.

`scripts/checkpoint_audit.py` kini menyertakan prompt checkpoint ID dan `canonicalOptions`, `visibleOptions`,
`hasTranslatedOptions`, sehingga audit berikutnya menampilkan teks yang benar-benar dilihat siswa.
Jalankan `E2E_LOCALE=id E2E_PORT=8092 LIVE_SPEECH=off WARM_SPEECH=off PREFETCH_CHECKPOINTS=off scripts/e2e.sh`
untuk mengulang pemeriksaan label Indonesia tanpa token OpenAI.

## Pembanding bunyi tidak boleh menjadi kunci — 18 Oktober 2026

Laporan pengguna: checkpoint AI bertanya `Kata mana yang punya bunyi pendek /ɪ/ seperti pada 'ship'?`, tetapi kuncinya `ship`.
Kata itu memang memiliki bunyi yang disebut, namun penyebutan `seperti ship` terbaca sebagai permintaan contoh lain dan sekaligus
membocorkan kunci. Perbaikan tidak menganggap jawaban pengguna `shap` otomatis benar; masalahnya ada pada rancangan soal.

- Penulis soal (termasuk latihan serupa dan checkpoint level) serta verifier mendapat aturan konkret: kata pembanding bukan kunci.
  Minta kata lain yang sudah diajarkan, misalnya `sit` untuk pembanding `ship`, dengan pilihan `sit / seat / food`; atau gunakan
  audio tanpa menulis kata yang harus didengar dalam prompt. Opsi pengenalan memakai kata nyata yang telah diajarkan; ejaan rekaan
  hanya digunakan sebagai contoh kesalahan pengucapan yang secara eksplisit diminta untuk diperbaiki.
- Validasi server menolak kata pembanding yang muncul sebagai kunci MCQ, salah satu accepted answer teks, atau jawaban multi.
  Pemeriksaan mengenali cue pembanding EN/ID, bukan sekadar melarang kata kunci muncul di prompt. Pertanyaan yang menyebut
  pilihan, membandingkan `full / fool` dengan `food`, atau meminta huruf yang namanya terdengar seperti `eye` tetap sah.
  Penggunaan biasa `as a capital letter` dan `would like a short answer` tidak dianggap pembanding bunyi.
- Penolakan menyertakan alasan spesifik untuk retry generator. Tes database membuktikan draft melingkar tidak mencapai verifier;
  soal retry yang mengganti kunci dengan `sit` diterima, atau fallback authored digunakan setelah dua draft gagal. Keduanya masih
  dapat dinilai 100 dan membuka progres melalui alur normal.
- Audit 7.219 entri authored (termasuk versi prompt EN/ID, materi, dan template latihan) tidak menemukan kasus ini.
  Scanner audit kini memberi flag `sound_comparison_reuses_reference_as_answer` untuk pemeriksaan berikutnya.
- Tes Go/database, coursecheck, dan e2e mock 194 node dalam mode Indonesia lolos. Tidak memakai token OpenAI atau TTS.

Riwayat attempt yang sudah dinilai dipertahankan. Aturan baru berlaku pada soal yang dibuat setelah kode diperbarui;
perbaikan ini belum di-commit, di-push, atau diterapkan ke produksi.
