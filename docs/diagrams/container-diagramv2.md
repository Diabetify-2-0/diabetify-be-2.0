# Dokumentasi Container Diagram Diabetify 2.0

## 1. Tujuan Diagram

Container Diagram berada satu level di bawah System Context Diagram. Jika System Context Diagram hanya menjelaskan hubungan antara **Sistem Diabetify** dengan aktor dan sistem eksternal, maka Container Diagram menjelaskan isi utama dari Sistem Diabetify.

Tujuan utama diagram ini:

- Menjelaskan container besar yang membentuk Sistem Diabetify 2.0.
- Menunjukkan bagaimana mobile app, control panel, backend, ML service, counterfactual planner, RAG service, governance service, dan data store saling berinteraksi.
- Menegaskan batas antara container internal Diabetify dan sistem eksternal seperti Google OAuth, SMTP Server, LLM Provider, dan Sumber Pedoman Medis.
- Menjelaskan bahwa Diabetify 2.0 menggunakan pendekatan modular service-based architecture, bukan monolith penuh dan bukan klaim microservice penuh.

## 2. Batas Sistem

Boundary utama pada diagram adalah:

`Sistem Diabetify 2.0`

Semua container di dalam boundary ini dianggap sebagai bagian dari solusi Diabetify 2.0. Container tersebut dapat berada pada repository atau service berbeda, tetapi secara arsitektur tetap merupakan bagian dari ekosistem internal Diabetify.

Container internal meliputi:

- Aplikasi Diabetify Mobile.
- Diabetify Control Panel.
- Diabetify Backend API.
- ML Prediction Service.
- Counterfactual Planner Service.
- XAI Chatbot / RAG Service.
- Continual Learning & Model Governance Service.
- Model Registry.
- PostgreSQL Database.
- RabbitMQ.
- Redis Cache.
- Vector Store / Knowledge Base.

Sistem eksternal yang berada di luar boundary:

- Google OAuth.
- SMTP Server.
- LLM Provider.
- Sumber Pedoman Medis.

Pemisahan ini penting agar diagram tidak mencampur antara bagian internal sistem dengan layanan pihak ketiga atau sumber eksternal.

## 3. Aktor Sistem

### 3.1 Pengguna

Pengguna adalah aktor utama yang memakai Aplikasi Diabetify Mobile. Pada Diabetify 2.0, pengguna tidak hanya memakai aplikasi untuk prediksi risiko, tetapi juga untuk memperoleh rekomendasi counterfactual dan berinteraksi dengan chatbot XAI.

Relasi pada diagram:

`Pengguna -> Aplikasi Diabetify Mobile`

Makna relasi:

- Pengguna mengakses layanan prediksi.
- Pengguna menerima rekomendasi counterfactual.
- Pengguna berinteraksi dengan chatbot XAI untuk memahami hasil prediksi.

### 3.2 Admin / Operator

Admin atau operator memakai Diabetify Control Panel untuk memantau operasional sistem. Peran ini tidak berfokus pada validasi model secara teknis atau medis, tetapi pada monitoring layanan dan aktivitas sistem.

Relasi pada diagram:

`Admin / Operator -> Diabetify Control Panel`

Makna relasi:

- Admin memantau status layanan.
- Admin memantau aktivitas sistem.
- Admin melihat kondisi operasional melalui control panel.

### 3.3 Data Scientist / ML Engineer

Data Scientist atau ML Engineer bertanggung jawab pada aspek teknis machine learning. Peran ini mengevaluasi drift, melakukan retraining, dan menyiapkan model challenger.

Relasi pada diagram:

`Data Scientist / ML Engineer -> Diabetify Control Panel`

Makna relasi:

- Data Scientist mengevaluasi drift.
- Data Scientist menjalankan atau meninjau proses retraining.
- Data Scientist menyiapkan model challenger untuk dievaluasi.

### 3.4 Medical Expert / Reviewer

Medical Expert atau Reviewer bertanggung jawab pada validasi klinis. Peran ini memastikan bahwa model challenger dan keputusan yang akan digunakan tetap layak secara medis.

Relasi pada diagram:

`Medical Expert / Reviewer -> Diabetify Control Panel`

Makna relasi:

- Medical Expert memvalidasi aspek medis model challenger.
- Medical Expert memberikan persetujuan klinis sebelum model digunakan.
- Medical Expert menjadi bagian dari mekanisme Human-in-the-Loop.

## 4. Kelompok Container Internal

Container internal pada diagram dibagi menjadi empat kelompok besar:

- User-Facing Containers.
- XAI & ML Service Containers.
- Model Governance Containers.
- Data & Infrastructure Containers.

Pembagian ini dibuat agar tanggung jawab setiap bagian sistem terlihat jelas.

## 5. User-Facing Containers

### 5.1 Aplikasi Diabetify Mobile

Container ini adalah aplikasi Android yang digunakan oleh pengguna akhir.

Teknologi:

- Android App.
- Kotlin.
- Jetpack Compose.

Tanggung jawab utama:

- Menyediakan antarmuka pengguna untuk autentikasi.
- Menampilkan profil dan aktivitas pengguna.
- Mengirim permintaan prediksi.
- Menampilkan hasil prediksi risiko diabetes.
- Menampilkan rekomendasi counterfactual.
- Menyediakan akses ke chatbot XAI.

Relasi utama:

`Mobile -> Backend : REST API requests`

Mobile App tidak berkomunikasi langsung dengan ML service, database, RabbitMQ, RAG service, atau service internal lain. Semua request pengguna diarahkan ke Backend API.

### 5.2 Diabetify Control Panel

Control Panel adalah aplikasi web untuk aktor internal seperti Admin, Data Scientist, dan Medical Expert.

Tanggung jawab utama:

- Menampilkan monitoring operasional sistem.
- Menampilkan laporan drift dan lifecycle model.
- Mendukung proses validasi model challenger.
- Mendukung approval atau rejection oleh Medical Expert.
- Menyediakan akses terhadap informasi model dan audit operasional.

Relasi utama:

`ControlPanel -> Backend : Internal user API requests`

Keputusan penting:

- Control Panel tidak mengakses PostgreSQL secara langsung.
- Control Panel tidak mengakses Model Registry secara langsung.
- Control Panel tidak memanggil Governance Service secara langsung.
- Semua akses dilakukan melalui Backend API agar kontrol otorisasi, audit, dan validasi tetap terpusat.

### 5.3 Diabetify Backend API

Backend API adalah pusat orkestrasi pada Sistem Diabetify 2.0. Backend menerima request dari Mobile App dan Control Panel, lalu meneruskan proses ke container lain sesuai kebutuhan.

Teknologi:

- Go.
- Gin.

Tanggung jawab utama:

- Autentikasi dan otorisasi.
- Pengelolaan user, profil, aktivitas, dan prediksi.
- Orkestrasi job prediksi.
- Orkestrasi job counterfactual.
- Integrasi dengan RAG Service untuk chatbot XAI.
- Integrasi dengan Governance Service untuk lifecycle model.
- Penyedia API untuk Control Panel.
- Pengelolaan knowledge base yang telah dikurasi.
- Integrasi dengan PostgreSQL, Redis, RabbitMQ, Google OAuth, SMTP, Model Registry, Vector Store, dan Sumber Pedoman Medis.

Relasi utama:

- `Backend -> Google OAuth` untuk verifikasi token Google.
- `Backend -> SMTP Server` untuk email verifikasi, reset password, dan notifikasi.
- `Backend -> PostgreSQL` untuk CRUD data operasional.
- `Backend -> Redis` untuk cache hasil sementara.
- `Backend -> RabbitMQ` untuk publish prediction dan counterfactual jobs.
- `RabbitMQ -> Backend` untuk deliver hasil prediction dan counterfactual.
- `Backend -> RAG Service` untuk request chatbot XAI dan explanation.
- `Backend -> Governance Service` untuk trigger lifecycle tasks, submit approvals, dan retrieve reports.
- `Backend -> Model Registry` untuk membaca metadata versi model.
- `Backend -> Vector Store` untuk mengelola knowledge base terkurasi.
- `Backend -> Sumber Pedoman Medis` untuk mengambil referensi medis.

## 6. XAI & ML Service Containers

### 6.1 ML Prediction Service

ML Prediction Service menjalankan model prediksi risiko diabetes. Service ini menjadi sumber utama untuk menghitung risk score dan menghasilkan informasi interpretabilitas model.

Teknologi:

- Python.
- XGBoost.
- SHAP.

Tanggung jawab utama:

- Menerima prediction request dari RabbitMQ.
- Menjalankan model prediksi.
- Menghasilkan risk score.
- Menghasilkan explanation metadata berbasis model.
- Mengirim prediction result kembali ke RabbitMQ.
- Menjadi evaluator skenario kandidat dari Counterfactual Planner.
- Digunakan oleh Governance Service untuk shadow validation.

Relasi utama:

- `RabbitMQ -> ML Service : Prediction request`.
- `ML Service -> RabbitMQ : Prediction result`.
- `Counterfactual Planner -> ML Service : Evaluate candidate scenarios`.
- `Governance Service -> ML Service : Run shadow prediction validation`.

### 6.2 Counterfactual Planner Service

Counterfactual Planner Service adalah service baru yang menghasilkan rekomendasi aksi berdasarkan pendekatan counterfactual. Service ini mencari skenario perubahan yang dapat menurunkan risiko pengguna dengan tetap mempertimbangkan batasan dan realisme perubahan.

Tanggung jawab utama:

- Menerima counterfactual planning request dari RabbitMQ.
- Menghasilkan kandidat perubahan fitur yang memungkinkan.
- Mengevaluasi kandidat menggunakan ML Prediction Service.
- Memfilter kandidat berdasarkan constraint, mutability, dan safety.
- Meranking skenario berdasarkan risk reduction, actionability, proximity, sparsity, dan plausibility.
- Mengirim hasil counterfactual kembali melalui RabbitMQ.

Relasi utama:

- `RabbitMQ -> Counterfactual Planner : Counterfactual planning request`.
- `Counterfactual Planner -> RabbitMQ : Counterfactual result`.
- `Counterfactual Planner -> ML Service : Evaluate candidate scenarios`.
- `Counterfactual Planner -> Model Registry : Read active model metadata`.

Keputusan penting:

- Counterfactual Planner tidak menggantikan model prediksi.
- Counterfactual Planner menggunakan ML Prediction Service sebagai evaluator risk score.
- Counterfactual Planner membaca metadata model aktif dari Model Registry agar rekomendasi selaras dengan model production.
- Counterfactual Planner tidak mengelola data operasional pengguna secara langsung.

### 6.3 XAI Chatbot / RAG Service

XAI Chatbot / RAG Service menyediakan pengalaman percakapan bagi pengguna. Service ini menggunakan pendekatan Retrieval-Augmented Generation agar jawaban tidak hanya berasal dari LLM, tetapi juga berdasarkan konteks medis yang tersimpan dalam knowledge base.

Tanggung jawab utama:

- Menerima pertanyaan pengguna dari Backend API.
- Mengambil konteks medis dari Vector Store / Knowledge Base.
- Membangun prompt berbasis konteks.
- Mengirim prompt ke LLM Provider.
- Menghasilkan jawaban XAI yang grounded.
- Menyimpan metadata percakapan ke PostgreSQL.

Relasi utama:

- `Backend -> RAG Service : XAI chat and explanation request`.
- `RAG Service -> Vector Store : Retrieve medical context`.
- `RAG Service -> LLM Provider : Generate grounded response`.
- `RAG Service -> PostgreSQL : Store conversation metadata`.

Keputusan penting:

- LLM Provider tidak dijadikan satu-satunya sumber kebenaran.
- Jawaban chatbot harus digrounding menggunakan konteks dari knowledge base.
- Metadata percakapan disimpan untuk audit, analisis, atau pengembangan fitur lanjutan.

## 7. Model Governance Containers

### 7.1 Continual Learning & Model Governance Service

Container ini menangani lifecycle model. Tujuannya adalah menjaga agar model tetap valid, aman, dan relevan ketika karakteristik data berubah.

Tanggung jawab utama:

- Mendeteksi data drift dan prediction drift.
- Menjalankan atau mengelola proses retraining.
- Mengevaluasi candidate model atau model challenger.
- Menjalankan shadow validation.
- Membandingkan model candidate dengan model production.
- Menyediakan laporan kepada Backend API untuk ditampilkan pada Control Panel.

Relasi utama:

- `Backend -> Governance Service : Trigger lifecycle tasks, submit approvals, retrieve reports`.
- `Governance Service -> PostgreSQL : Read historical data, feedback, and validation metrics`.
- `Governance Service -> Model Registry : Store and compare model versions`.
- `Governance Service -> ML Service : Run shadow prediction validation`.

Keputusan penting:

- Governance tidak dipaksa berjalan melalui RabbitMQ pada rancangan ini.
- Backend API dapat memicu lifecycle task dan mengambil laporan melalui Governance Service.
- Approval manusia tetap dilakukan melalui Control Panel dan diteruskan oleh Backend API.

### 7.2 Model Registry

Model Registry adalah data store khusus untuk menyimpan dan mengelola versi model.

Tanggung jawab utama:

- Menyimpan metadata model production.
- Menyimpan candidate model atau model challenger.
- Menyimpan archived model.
- Menyimpan status model seperti production, candidate, approved, rejected, atau archived.
- Mendukung proses evaluasi, comparison, dan rollback.

Relasi utama:

- `Backend -> Model Registry : Review model versions and active model metadata`.
- `Counterfactual Planner -> Model Registry : Read active model metadata`.
- `Governance Service -> Model Registry : Store and compare model versions`.

Keputusan penting:

- Model Registry dipisahkan dari PostgreSQL operasional karena tanggung jawabnya berbeda.
- PostgreSQL menyimpan data operasional aplikasi, sedangkan Model Registry menyimpan lifecycle model.

## 8. Data & Infrastructure Containers

### 8.1 PostgreSQL Database

PostgreSQL adalah data store utama untuk data operasional sistem.

Data yang disimpan:

- Data pengguna.
- Profil pengguna.
- Aktivitas pengguna.
- Prediction jobs dan hasil prediksi.
- Counterfactual jobs dan hasil rekomendasi.
- Feedback pengguna.
- Chat history atau metadata percakapan.
- Validation metrics.
- Audit logs.

Relasi utama:

- `Backend -> PostgreSQL : CRUD data operasional`.
- `RAG Service -> PostgreSQL : Store conversation metadata`.
- `Governance Service -> PostgreSQL : Read historical data, feedback, and validation metrics`.

### 8.2 RabbitMQ

RabbitMQ adalah message broker untuk proses asynchronous. Pada rancangan final ini, RabbitMQ difokuskan untuk dua proses:

- Prediction.
- Counterfactual.

Tanggung jawab utama:

- Menerima prediction jobs dari Backend.
- Mengirim prediction request ke ML Prediction Service.
- Mengembalikan prediction result ke Backend.
- Menerima counterfactual jobs dari Backend.
- Mengirim counterfactual planning request ke Counterfactual Planner Service.
- Mengembalikan counterfactual result ke Backend.

Relasi utama:

- `Backend -> RabbitMQ : Publish prediction and counterfactual jobs`.
- `RabbitMQ -> Backend : Deliver prediction and counterfactual results`.
- `RabbitMQ -> ML Service : Prediction request`.
- `ML Service -> RabbitMQ : Prediction result`.
- `RabbitMQ -> Counterfactual Planner : Counterfactual planning request`.
- `Counterfactual Planner -> RabbitMQ : Counterfactual result`.

Keputusan penting:

- RabbitMQ tidak digunakan untuk governance job pada rancangan final.
- Governance dipicu melalui Backend API dan Governance Service agar diagram lebih sederhana dan tidak over-engineered.
- Prediction dan counterfactual tetap asynchronous karena keduanya berpotensi memerlukan waktu pemrosesan lebih lama dibanding request API biasa.

### 8.3 Redis Cache

Redis digunakan sebagai cache untuk data sementara.

Tanggung jawab utama:

- Menyimpan hasil sementara.
- Menyimpan session cache.
- Mengurangi beban PostgreSQL untuk data yang bersifat ephemeral.
- Mempercepat akses terhadap data yang sering dibaca tetapi tidak selalu perlu disimpan permanen.

Relasi utama:

- `Backend -> Redis : Cache hasil sementara`.

### 8.4 Vector Store / Knowledge Base

Vector Store / Knowledge Base menyimpan potongan dokumen medis dan embedding yang digunakan oleh RAG Service.

Tanggung jawab utama:

- Menyimpan medical guideline chunks.
- Menyimpan embeddings.
- Mendukung semantic retrieval.
- Menjadi sumber grounding untuk jawaban chatbot XAI.
- Menyimpan knowledge base yang telah dikurasi.

Relasi utama:

- `Backend -> Vector Store : Manage curated medical knowledge base`.
- `RAG Service -> Vector Store : Retrieve medical context`.

Keputusan penting:

- Vector Store tidak langsung diakses oleh Mobile App atau Control Panel.
- Kurasi knowledge base tetap melalui Backend API.
- RAG Service hanya mengambil konteks yang relevan untuk menjawab pertanyaan pengguna.

## 9. Sistem Eksternal

### 9.1 Google OAuth

Google OAuth digunakan untuk memverifikasi token autentikasi Google.

Relasi utama:

`Backend -> Google OAuth : Verifikasi token Google`

Google OAuth berada di luar boundary Diabetify karena merupakan layanan pihak ketiga.

### 9.2 SMTP Server

SMTP Server digunakan untuk mengirim email.

Kebutuhan email:

- Email verifikasi.
- Reset password.
- Notifikasi sistem.

Relasi utama:

`Backend -> SMTP Server : Email verifikasi, reset password, dan notifikasi`

### 9.3 LLM Provider

LLM Provider digunakan oleh RAG Service untuk menghasilkan jawaban percakapan.

Relasi utama:

`RAG Service -> LLM Provider : Generate grounded response`

Keputusan penting:

- LLM Provider tidak menerima request langsung dari Mobile App.
- LLM Provider tidak dipakai tanpa konteks.
- Jawaban yang dihasilkan harus berbasis konteks dari Vector Store.

### 9.4 Sumber Pedoman Medis

Sumber Pedoman Medis adalah sumber eksternal yang menyediakan referensi untuk knowledge base.

Contoh sumber:

- Pedoman klinis.
- Dokumen medis resmi.
- Referensi edukasi kesehatan yang tervalidasi.

Relasi utama:

`Backend -> Sumber Pedoman Medis : Retrieve reference documents for knowledge-base curation`

Keputusan penting:

- Sumber Pedoman Medis tidak langsung masuk ke RAG Service.
- Referensi harus melalui proses kurasi sebelum masuk ke Vector Store.
- Kurasi dilakukan melalui mekanisme internal sistem, terutama lewat Backend API dan Control Panel.

## 10. Alur Utama Sistem

### 10.1 Alur Prediksi Risiko

1. Pengguna menggunakan Mobile App.
2. Mobile App mengirim request prediksi ke Backend API.
3. Backend API menyimpan atau membaca data yang diperlukan dari PostgreSQL.
4. Backend API mempublish prediction job ke RabbitMQ.
5. RabbitMQ meneruskan prediction request ke ML Prediction Service.
6. ML Prediction Service menjalankan model prediksi.
7. ML Prediction Service mengirim prediction result ke RabbitMQ.
8. RabbitMQ mengirim hasil prediksi ke Backend API.
9. Backend API menyimpan hasil prediksi dan mengembalikannya ke Mobile App.

### 10.2 Alur Counterfactual

1. Pengguna meminta rekomendasi counterfactual melalui Mobile App.
2. Mobile App mengirim request ke Backend API.
3. Backend API mempublish counterfactual job ke RabbitMQ.
4. RabbitMQ meneruskan planning request ke Counterfactual Planner Service.
5. Counterfactual Planner membangun kandidat skenario.
6. Counterfactual Planner mengevaluasi kandidat menggunakan ML Prediction Service.
7. Counterfactual Planner membaca metadata model aktif dari Model Registry.
8. Counterfactual Planner memilih rekomendasi yang feasible, actionable, dan aman.
9. Counterfactual Planner mengirim hasil ke RabbitMQ.
10. RabbitMQ mengirim hasil ke Backend API.
11. Backend API menyimpan atau menyajikan hasil rekomendasi ke Mobile App.

### 10.3 Alur Chatbot XAI berbasis RAG

1. Pengguna mengirim pertanyaan melalui Mobile App.
2. Mobile App mengirim request ke Backend API.
3. Backend API meneruskan request ke RAG Service.
4. RAG Service mengambil konteks dari Vector Store.
5. RAG Service membangun prompt berbasis konteks.
6. RAG Service mengirim prompt ke LLM Provider.
7. LLM Provider menghasilkan jawaban.
8. RAG Service menyimpan metadata percakapan ke PostgreSQL.
9. Jawaban dikembalikan ke Backend API dan ditampilkan di Mobile App.

### 10.4 Alur Continual Learning dan Model Governance

1. Backend API memicu lifecycle task atau Governance Service menjalankan proses berdasarkan jadwal internal.
2. Governance Service membaca data historis, feedback, dan validation metrics dari PostgreSQL.
3. Governance Service menjalankan drift detection.
4. Governance Service menjalankan retraining atau evaluasi model challenger.
5. Governance Service menjalankan shadow validation menggunakan ML Prediction Service.
6. Governance Service menyimpan dan membandingkan versi model di Model Registry.
7. Backend API mengambil laporan dari Governance Service.
8. Data Scientist meninjau drift, retraining, dan model challenger melalui Control Panel.
9. Medical Expert memvalidasi aspek medis model challenger dan memberi persetujuan klinis.
10. Admin memantau status layanan dan aktivitas sistem melalui Control Panel.

### 10.5 Alur Knowledge Base RAG

1. Backend API mengambil referensi dari Sumber Pedoman Medis.
2. Referensi dikurasi melalui mekanisme internal sistem.
3. Backend API mengelola dokumen terkurasi ke Vector Store.
4. Vector Store menyimpan chunks dan embeddings.
5. RAG Service mengambil konteks dari Vector Store ketika menjawab pertanyaan pengguna.

## 11. Justifikasi Arsitektur

Rancangan Diabetify 2.0 menggunakan pendekatan **modular service-based architecture**. Pendekatan ini dipilih karena sistem memiliki beberapa kemampuan besar yang karakteristiknya berbeda.

Alasan pemisahan container:

- ML Prediction Service dipisahkan karena inference model memiliki dependency dan runtime berbeda dari Backend API.
- Counterfactual Planner dipisahkan karena proses optimasi dapat berat dan membutuhkan aturan domain khusus.
- RAG Service dipisahkan karena memiliki alur retrieval, prompt construction, dan LLM integration tersendiri.
- Governance Service dipisahkan karena lifecycle model, drift detection, retraining, dan shadow validation memiliki kompleksitas tersendiri.
- Model Registry dipisahkan karena metadata model dan status lifecycle berbeda dari data operasional aplikasi.
- Vector Store dipisahkan karena pencarian berbasis embedding memiliki kebutuhan data store yang berbeda dari PostgreSQL.

Alasan Backend tetap menjadi pusat orkestrasi:

- Mobile App dan Control Panel memiliki satu pintu API yang konsisten.
- Autentikasi, otorisasi, validasi, dan audit lebih mudah dikontrol.
- Control Panel tidak perlu mengetahui detail internal database, registry, vector store, atau service lain.
- Perubahan internal service dapat dilakukan tanpa mengubah kontrak aplikasi pengguna secara langsung.

Alasan RabbitMQ dibatasi untuk prediction dan counterfactual:

- Prediction dan counterfactual adalah proses yang berpotensi membutuhkan waktu lebih lama dibanding request API biasa.
- Keduanya cocok diproses secara asynchronous agar request pengguna tidak menunggu proses komputasi berat secara sinkron.
- RabbitMQ membantu memisahkan Backend API dari service komputasi seperti ML Prediction Service dan Counterfactual Planner Service.
- Governance tidak dipaksa melalui RabbitMQ agar rancangan tidak terlalu kompleks.
- Lifecycle governance dapat dipicu langsung oleh Backend API atau scheduler internal pada Model Governance Service.

## 12. Analisis Arsitektur Container

Secara arsitektur, Container Diagram Diabetify 2.0 sudah menunjukkan rancangan yang modular, terarah, dan cukup bersih. Sistem tidak lagi diposisikan sebagai satu backend besar yang menanggung semua tanggung jawab, melainkan dibagi berdasarkan domain utama: user-facing API, prediksi ML, counterfactual planning, chatbot RAG, model governance, dan data infrastructure.

Keputusan paling penting pada diagram ini adalah menjadikan **Backend API sebagai orchestration layer**. Artinya, Backend API bukan tempat semua logic berat dijalankan, tetapi menjadi pintu masuk utama yang mengatur autentikasi, otorisasi, request routing, penyimpanan operasional, dan komunikasi dengan service lain.

Pendekatan ini baik karena Mobile App dan Control Panel tidak perlu mengetahui detail internal sistem. Mobile App cukup mengenal Backend API. Control Panel juga cukup mengenal Backend API. Dengan demikian, perubahan pada ML Service, RAG Service, Model Registry, atau Vector Store tidak langsung memengaruhi aplikasi pengguna.

## 13. Analisis Berdasarkan Area Sistem

### 13.1 User-Facing Layer

User-facing layer terdiri dari Mobile App, Control Panel, dan Backend API.

Analisis:

- Mobile App difokuskan untuk pengguna akhir.
- Control Panel difokuskan untuk aktor internal.
- Backend API menjadi pemisah antara antarmuka pengguna dan service internal.

Keputusan ini sudah tepat karena Mobile App dan Control Panel memiliki kebutuhan, hak akses, dan pola penggunaan yang berbeda. Pengguna akhir membutuhkan fitur prediksi, counterfactual, dan chatbot. Admin, Data Scientist, dan Medical Expert membutuhkan monitoring, validasi, approval, dan laporan model.

Jika Control Panel langsung mengakses database atau Model Registry, sistem akan lebih sulit diaudit dan lebih berisiko dari sisi keamanan. Karena itu, keputusan `ControlPanel -> Backend` adalah keputusan yang clean dan aman.

### 13.2 Prediction Flow

Prediction flow menggunakan Backend API, RabbitMQ, dan ML Prediction Service.

Analisis:

- Backend API tidak menjalankan inference secara langsung.
- RabbitMQ digunakan sebagai message broker.
- ML Prediction Service menjadi container khusus untuk model inference.

Desain ini tepat karena inference model memiliki karakteristik berbeda dari request CRUD biasa. Model inference dapat memiliki dependency Python, library ML, model artifact, dan kebutuhan runtime yang tidak cocok dicampur langsung ke Backend Go.

Dengan memisahkan ML Prediction Service, sistem menjadi lebih mudah dikembangkan. Backend dapat tetap fokus sebagai API, sedangkan ML Service fokus menjalankan model.

### 13.3 Counterfactual Flow

Counterfactual flow menggunakan Backend API, RabbitMQ, Counterfactual Planner Service, ML Prediction Service, dan Model Registry.

Analisis:

- Counterfactual Planner dipisahkan sebagai service sendiri karena prosesnya berat dan domain-specific.
- Counterfactual Planner tidak menggantikan ML Prediction Service.
- Counterfactual Planner menggunakan ML Prediction Service untuk mengevaluasi kandidat skenario.
- Counterfactual Planner membaca metadata model aktif dari Model Registry.

Desain ini clean karena tugas setiap container jelas. Counterfactual Planner menghasilkan kandidat perubahan dan rekomendasi aksi. ML Prediction Service tetap menjadi sumber scoring risiko. Model Registry menjadi sumber metadata model aktif.

Counterfactual tidak langsung menulis ke PostgreSQL. Ini juga keputusan yang baik karena ownership data operasional tetap berada pada Backend API. Counterfactual Planner cukup mengirim hasil terstruktur melalui RabbitMQ, lalu Backend API yang menyimpan atau menyajikan hasil ke pengguna.

### 13.4 Counterfactual Explanation Flow

Counterfactual Planner menghasilkan hasil numerik dan rekomendasi terstruktur. Namun, angka minimum saja tidak cukup untuk pengguna. Pengguna tetap membutuhkan narasi yang mudah dipahami.

Alur yang paling bersih adalah:

1. Counterfactual Planner menghasilkan structured result.
2. Result dikirim ke Backend API melalui RabbitMQ.
3. Backend API mengirim structured result tersebut ke RAG Service jika dibutuhkan narasi.
4. RAG Service mengambil konteks medis dari Vector Store.
5. RAG Service membangun prompt berbasis hasil counterfactual dan konteks medis.
6. RAG Service memanggil LLM Provider.
7. LLM Provider menghasilkan narasi yang grounded.
8. Backend API mengirim angka dan narasi ke Mobile App.

Alur ini lebih baik daripada `Counterfactual Planner -> LLM Provider` secara langsung.

Alasannya:

- Counterfactual Planner tetap fokus pada optimasi dan perhitungan.
- RAG Service tetap fokus pada narasi, retrieval, dan grounding.
- LLM tidak menerima angka mentah tanpa konteks medis.
- Prompting dan LLM integration tidak tersebar di banyak service.
- Backend tetap menjadi orkestrator utama.

Dengan demikian, Counterfactual Planner menghasilkan **apa rekomendasinya**, sedangkan RAG Service membantu menjelaskan **mengapa rekomendasi itu masuk akal dan bagaimana pengguna memahaminya**.

### 13.5 RAG Chatbot Flow

RAG Chatbot menggunakan Backend API, RAG Service, Vector Store, LLM Provider, dan PostgreSQL.

Analisis:

- Backend API meneruskan pertanyaan dan konteks pengguna ke RAG Service.
- RAG Service mengambil konteks dari Vector Store.
- LLM Provider digunakan setelah konteks medis tersedia.
- Metadata percakapan disimpan ke PostgreSQL.

Desain ini mengikuti prinsip grounded generation. LLM tidak dipakai sebagai sumber kebenaran tunggal. Jawaban harus ditopang oleh knowledge base yang telah dikurasi.

Ini penting untuk domain kesehatan karena jawaban chatbot tidak boleh hanya mengandalkan generasi bebas dari model bahasa. Dengan RAG, jawaban dapat diarahkan agar tetap mengacu pada referensi medis yang telah disiapkan.

### 13.6 Model Governance Flow

Model Governance Service menangani drift detection, retraining, shadow validation, dan lifecycle model.

Analisis:

- Governance dipisahkan dari Backend API karena workflow model lifecycle cukup kompleks.
- Governance membaca data historis dari PostgreSQL.
- Governance menyimpan dan membandingkan model melalui Model Registry.
- Governance menggunakan ML Prediction Service untuk shadow validation.
- Backend API menjadi jalur untuk trigger task, submit approval, dan retrieve report.

Keputusan untuk tidak memasukkan governance ke RabbitMQ pada diagram final sudah masuk akal. Governance dapat memiliki scheduler internal atau dipicu langsung melalui Backend API. Dengan demikian, RabbitMQ tidak dipaksa menangani semua proses asynchronous di sistem.

Desain ini lebih sederhana dan lebih mudah dijelaskan. RabbitMQ fokus untuk prediction dan counterfactual, sementara governance memiliki lifecycle flow sendiri.

### 13.7 Knowledge Base Flow

Knowledge base flow melibatkan Backend API, Sumber Pedoman Medis, Vector Store, RAG Service, dan Control Panel.

Analisis:

- Backend API mengambil referensi dari Sumber Pedoman Medis.
- Referensi tidak langsung masuk ke RAG Service.
- Referensi harus dikurasi sebelum menjadi knowledge base.
- Vector Store menyimpan chunks dan embeddings.
- RAG Service mengambil konteks dari Vector Store saat menjawab pertanyaan.

Keputusan ini penting karena sumber medis harus dikontrol kualitasnya. Jika dokumen eksternal langsung masuk ke RAG tanpa kurasi, risiko jawaban tidak relevan atau tidak aman meningkat.

Dengan menempatkan Backend API sebagai pengelola knowledge base, sistem tetap memiliki titik kontrol untuk validasi, audit, dan update referensi.

## 14. Analisis Trade-off

Rancangan ini memiliki beberapa trade-off yang perlu dipahami.

### 14.1 Keuntungan

- Sistem lebih modular dan mudah dipahami.
- Backend API tidak menanggung logic ML, RAG, counterfactual, dan governance secara langsung.
- Setiap service memiliki tanggung jawab yang jelas.
- Proses berat seperti prediction dan counterfactual dapat berjalan asynchronous.
- Control Panel lebih aman karena tidak langsung mengakses database atau registry.
- RAG lebih aman karena menggunakan knowledge base medis.
- Governance mendukung validasi jangka panjang terhadap model.

### 14.2 Konsekuensi

- Jumlah container bertambah.
- Komunikasi antar-service menjadi lebih kompleks.
- Dibutuhkan error handling antar-service yang baik.
- Dibutuhkan observability agar job asynchronous bisa dilacak.
- Dibutuhkan versioning untuk model, prompt, knowledge base, dan API contract.
- Dibutuhkan strategi fallback jika ML Service, RAG Service, LLM Provider, atau RabbitMQ bermasalah.

Trade-off ini wajar. Rancangan Diabetify 2.0 lebih kompleks daripada Diabetify 1.0 karena target fiturnya juga lebih kompleks.

## 15. Risiko Arsitektur dan Mitigasi

### 15.1 Risiko RabbitMQ menjadi bottleneck

RabbitMQ menangani prediction dan counterfactual. Jika volume job tinggi, queue dapat menumpuk.

Mitigasi:

- Gunakan retry policy.
- Gunakan dead-letter queue.
- Pisahkan queue prediction dan counterfactual.
- Tambahkan monitoring queue length dan processing time.
- Scale worker ML atau Counterfactual Planner jika diperlukan.

### 15.2 Risiko LLM menghasilkan jawaban tidak akurat

LLM Provider dapat menghasilkan jawaban yang kurang tepat jika prompt atau konteks tidak cukup baik.

Mitigasi:

- Gunakan RAG dengan sumber medis yang dikurasi.
- Batasi jawaban agar mengikuti konteks yang tersedia.
- Simpan metadata percakapan untuk audit.
- Sediakan disclaimer bahwa jawaban bukan pengganti diagnosis dokter.
- Libatkan Medical Expert untuk validasi knowledge base.

### 15.3 Risiko rekomendasi counterfactual tidak realistis

Counterfactual yang hanya mengejar penurunan risiko matematis dapat menghasilkan saran yang tidak realistis.

Mitigasi:

- Gunakan feature mutability rules.
- Gunakan lifestyle constraint rules.
- Gunakan medical safety rules.
- Ranking skenario berdasarkan actionability, proximity, sparsity, dan plausibility.
- Berikan beberapa opsi rekomendasi, bukan hanya satu angka minimum.

### 15.4 Risiko model drift tidak terdeteksi

Jika data pengguna berubah seiring waktu, performa model dapat menurun.

Mitigasi:

- Jalankan drift detection secara berkala.
- Simpan feedback dan validation metrics.
- Gunakan Model Governance Service untuk evaluasi model.
- Terapkan shadow validation sebelum model challenger dipromosikan.

### 15.5 Risiko akses internal terlalu luas

Control Panel digunakan oleh beberapa peran internal dengan tanggung jawab berbeda.

Mitigasi:

- Gunakan role-based access control.
- Admin, Data Scientist, dan Medical Expert harus memiliki hak akses berbeda.
- Semua request Control Panel harus melalui Backend API.
- Simpan audit log untuk aksi penting seperti approval model.

## 16. Pertanyaan Desain dan Jawaban Arsitektural

### 16.1 Mengapa Backend API menjadi pusat orkestrasi?

Karena Backend API adalah satu titik kontrol untuk autentikasi, otorisasi, validasi request, audit, dan kontrak API. Jika Mobile App atau Control Panel langsung mengakses service internal, sistem akan lebih sulit dikontrol dan lebih rentan inkonsistensi.

### 16.2 Mengapa Counterfactual Planner tidak langsung ke LLM?

Karena Counterfactual Planner sebaiknya fokus pada optimasi dan structured recommendation. Narasi berbasis bahasa natural lebih tepat ditangani RAG Service karena RAG Service memiliki akses ke Vector Store dan LLM Provider.

Alur yang benar:

`Counterfactual Planner -> RabbitMQ -> Backend API -> RAG Service -> LLM Provider`

### 16.3 Mengapa RabbitMQ tidak digunakan untuk governance?

Karena governance tidak harus selalu mengikuti pola request-result seperti prediction dan counterfactual. Governance dapat dipicu oleh Backend API atau scheduler internal. Menggunakan RabbitMQ untuk governance pada tahap desain ini akan membuat diagram lebih kompleks tanpa kebutuhan yang benar-benar kuat.

### 16.4 Mengapa Control Panel tidak langsung ke Model Registry?

Karena akses langsung dari Control Panel ke Model Registry akan melemahkan kontrol otorisasi dan audit. Backend API harus menjadi perantara agar akses model version, approval, rejection, dan reporting tetap tervalidasi.

### 16.5 Mengapa Model Registry dipisahkan dari PostgreSQL?

Karena Model Registry memiliki tanggung jawab khusus untuk lifecycle model. PostgreSQL menyimpan data operasional aplikasi, sedangkan Model Registry menyimpan model artifact, metadata versi, status model, dan riwayat lifecycle.

### 16.6 Mengapa Vector Store dipisahkan dari PostgreSQL?

Karena Vector Store digunakan untuk semantic retrieval berbasis embedding. Kebutuhannya berbeda dari relational query biasa. PostgreSQL tetap cocok untuk data operasional, tetapi knowledge base RAG lebih cocok ditempatkan pada vector store.

### 16.7 Apakah rancangan ini microservice?

Rancangan ini lebih tepat disebut **modular service-based architecture**. Ia memiliki beberapa service terpisah, tetapi belum otomatis berarti microservice penuh. Untuk disebut microservice penuh, perlu ada independensi deployment, scaling, observability, ownership, dan data ownership yang lebih ketat.

## 17. Konsistensi dengan Diagram Lain

Container Diagram V2 konsisten dengan System Context V2 karena:

- Aktor eksternal tetap Pengguna, Admin, Data Scientist, dan Medical Expert.
- Sistem eksternal tetap Google OAuth, SMTP Server, LLM Provider, dan Sumber Pedoman Medis.
- Container internal seperti Backend API, ML Service, RabbitMQ, PostgreSQL, Redis, RAG Service, Model Registry, dan Vector Store tetap berada di dalam Sistem Diabetify.

Container Diagram V2 konsisten dengan Backend Component Diagram V2 karena:

- Backend menjadi pusat orkestrasi.
- Prediction dan counterfactual dikirim melalui RabbitMQ.
- RAG, Governance, Model Registry, Vector Store, dan Medical Sources diakses melalui client/orchestrator backend.
- Control Panel tidak melakukan bypass ke database atau service internal.

Container Diagram V2 konsisten dengan Counterfactual Component Diagram V2 karena:

- Counterfactual Planner menerima job dari RabbitMQ.
- Counterfactual Planner mengirim hasil ke RabbitMQ.
- Counterfactual Planner menggunakan ML Prediction Service untuk mengevaluasi skenario.
- Counterfactual Planner membaca metadata model dari Model Registry.
- Counterfactual Planner tidak menulis langsung ke PostgreSQL operasional.

## 18. Hal yang Sengaja Tidak Ditampilkan

Beberapa hal tidak ditampilkan dalam Container Diagram V2 karena berada di luar fokus diagram:

- Detail endpoint API.
- Detail class, function, atau method.
- Detail deployment seperti VPS, container runtime, domain, SSL, dan load balancer.
- Detail sequence request per fitur.
- Detail schema database atau ERD.
- Detail internal algoritma counterfactual.
- Detail internal prompt engineering RAG.
- Detail pipeline training secara step-by-step.

Hal-hal tersebut dapat dijelaskan melalui diagram lain jika diperlukan, tetapi tidak wajib untuk Container Diagram.

## 19. Kesimpulan Analisis

Container Diagram Diabetify 2.0 sudah menunjukkan rancangan arsitektur yang bersih, modular, dan dapat dipertanggungjawabkan. Setiap container memiliki tanggung jawab yang jelas dan tidak saling mencampur domain secara berlebihan.

Backend API berperan sebagai orchestration layer. ML Prediction Service berfokus pada inference. Counterfactual Planner berfokus pada rekomendasi aksi. RAG Service berfokus pada penjelasan natural berbasis konteks medis. Governance Service berfokus pada lifecycle model. Model Registry dan Vector Store dipisahkan karena keduanya memiliki kebutuhan penyimpanan yang berbeda dari data operasional biasa.

Secara software engineering, rancangan ini kuat karena menerapkan separation of concerns, loose coupling, asynchronous processing untuk proses berat, human-in-the-loop untuk proses kritis, dan grounding berbasis knowledge base untuk jawaban XAI.

Dengan demikian, Container Diagram V2 dapat digunakan sebagai dasar dokumentasi arsitektur Diabetify 2.0 dan sebagai acuan untuk pengembangan lanjutan.
