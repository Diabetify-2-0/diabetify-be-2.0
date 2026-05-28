# Integrasi Counterfactual Planner dengan LLM/RAG

Dokumen ini menjelaskan rencana integrasi antara hasil counterfactual dari `diabetify-cf`, backend `diabetify-be`, dan modul chatbot/LLM/RAG.

## Tujuan

Kita ingin memisahkan tanggung jawab secara clean:

- `diabetify-cf` menghasilkan counterfactual terstruktur.
- Modul chatbot/RAG menghasilkan narasi planner berbasis LLM dan knowledge base.
- `diabetify-be` menjadi orchestrator, menyimpan hasil, dan menyediakan data ke mobile.
- `diabetify-mobile` hanya menampilkan hasil final dan menjalankan UX planner.

Dengan desain ini, API key LLM cukup berada di modul chatbot/RAG. `diabetify-cf` tidak perlu memanggil LLM secara langsung.

## Arsitektur Target

```text
Mobile
  -> diabetify-be
      -> diabetify-cf
      -> chatbot/RAG service
      -> database
  -> Mobile
```

Flow utamanya:

1. Mobile meminta counterfactual ke `diabetify-be`.
2. `diabetify-be` membuat counterfactual job.
3. `diabetify-be` mengirim job ke `diabetify-cf`.
4. `diabetify-cf` menghasilkan counterfactual result terstruktur.
5. `diabetify-be` menerima result dari `diabetify-cf`.
6. Jika result feasible, `diabetify-be` mengirim context ke chatbot/RAG.
7. Chatbot/RAG menghasilkan planner narrative.
8. `diabetify-be` merge counterfactual result + planner narrative.
9. `diabetify-be` menyimpan final result ke `counterfactual_jobs.result_payload`.
10. Mobile mengambil result dari endpoint counterfactual yang sama seperti sekarang.

## Pembagian Tanggung Jawab

### diabetify-cf

Tanggung jawab:

- Generate counterfactual candidates.
- Menentukan feasibility.
- Menghasilkan prediction before/after.
- Menghasilkan changed features.
- Menghasilkan `planner_input` terstruktur.
- Menyediakan template planner sebagai fallback jika dibutuhkan.

Tidak bertanggung jawab untuk:

- Memanggil LLM utama.
- RAG/knowledge base.
- Chatbot conversation.
- Menyimpan goal/check-in user.

### chatbot/RAG service

Tanggung jawab:

- Menyimpan/mengelola API key LLM.
- Mengambil grounding dari knowledge base medis.
- Menghasilkan planner narrative dari hasil counterfactual.
- Menjawab pertanyaan user lewat chatbot.
- Mengembalikan output structured JSON, bukan hanya teks bebas.

### diabetify-be

Tanggung jawab:

- Auth/user ownership.
- Orchestrate call ke `diabetify-cf`.
- Orchestrate call ke chatbot/RAG planner endpoint.
- Menyimpan result counterfactual final.
- Menyimpan planner goal dan check-in.
- Fallback ke template planner jika RAG gagal.
- Menyediakan endpoint ke mobile.

### diabetify-mobile

Tanggung jawab:

- Menampilkan hasil counterfactual.
- Save as goal.
- Planner detail, progress, check-in, milestone, coach note, completion, history.
- Tidak memanggil RAG service langsung.

## Endpoint yang Dibutuhkan dari Chatbot/RAG

Minimal perlu endpoint:

```http
POST /rag/planner
```

Endpoint ini menerima context counterfactual dan mengembalikan planner narrative dalam format JSON terstruktur.

### Request dari diabetify-be ke Chatbot/RAG

Contoh request:

```json
{
  "request_id": "cf-job-id-123",
  "language": "id",
  "prediction_context": {
    "risk_score": 0.68,
    "risk_percentage": 68.0,
    "risk_category": "high_risk",
    "top_factors": [
      {
        "name": "BMI",
        "label": "Indeks Massa Tubuh",
        "contribution": 0.24,
        "explanation": "BMI memberikan kontribusi besar terhadap peningkatan risiko."
      },
      {
        "name": "moderate_physical_activity_frequency",
        "label": "Aktivitas Fisik",
        "contribution": 0.18,
        "explanation": "Frekuensi aktivitas fisik rendah meningkatkan risiko."
      }
    ]
  },
  "counterfactual_result": {
    "status": "FEASIBLE",
    "current_risk_percentage": 68.0,
    "projected_risk_percentage": 44.0,
    "target_risk_percentage": 45,
    "recommended_candidate_id": "candidate-1",
    "changed_features": [
      {
        "feature_name": "BMI",
        "label": "Indeks Massa Tubuh",
        "baseline_value": 29.2,
        "candidate_value": 26.5,
        "baseline_text": "29.2 kg/m2",
        "candidate_text": "26.5 kg/m2",
        "delta": -2.7
      },
      {
        "feature_name": "moderate_physical_activity_frequency",
        "label": "Aktivitas Fisik",
        "baseline_value": 1,
        "candidate_value": 4,
        "baseline_text": "1 hari/minggu",
        "candidate_text": "4 hari/minggu",
        "delta": 3
      }
    ]
  },
  "constraints": {
    "mutable_allowed": [
      "BMI",
      "moderate_physical_activity_frequency"
    ],
    "immutable_features": [
      "age"
    ],
    "must_not_change": []
  },
  "safety_context": {
    "medical_disclaimer_required": true,
    "clinical_review_features": [
      "is_hypertension",
      "is_cholesterol"
    ]
  }
}
```

Field yang wajib:

- `request_id`
- `language`
- `counterfactual_result.status`
- `counterfactual_result.current_risk_percentage`
- `counterfactual_result.projected_risk_percentage`
- `counterfactual_result.target_risk_percentage`
- `counterfactual_result.changed_features`
- `constraints.mutable_allowed`

Field yang sangat disarankan:

- `prediction_context.top_factors`
- `constraints.immutable_features`
- `safety_context`

## Response yang Dibutuhkan dari Chatbot/RAG

Response harus structured JSON.

Contoh response sukses:

```json
{
  "status": "success",
  "data": {
    "summary": "Rencana ini berfokus pada penurunan BMI secara bertahap dan peningkatan aktivitas fisik untuk membantu menurunkan risiko ke arah target.",
    "goals": [
      "Menurunkan BMI dari 29.2 kg/m2 menuju 26.5 kg/m2 secara bertahap.",
      "Meningkatkan aktivitas fisik dari 1 hari menjadi 4 hari per minggu."
    ],
    "action_steps": [
      "Mulai dengan aktivitas intensitas sedang 20-30 menit sebanyak 2-3 hari per minggu, lalu tingkatkan secara bertahap.",
      "Pantau berat badan seminggu sekali agar perubahan BMI dapat dievaluasi tanpa terlalu dipengaruhi fluktuasi harian.",
      "Fokus pada perubahan makan dan aktivitas yang konsisten, bukan penurunan berat badan yang terlalu cepat."
    ],
    "safety_notes": [
      "Jika memiliki keluhan nyeri dada, sesak, pusing berat, atau riwayat penyakit jantung, konsultasikan dengan tenaga kesehatan sebelum meningkatkan intensitas olahraga.",
      "Target ini bersifat pendamping keputusan dan tidak menggantikan diagnosis atau terapi dari tenaga kesehatan."
    ],
    "monitoring_plan": [
      "Check-in berat badan setiap minggu.",
      "Catat aktivitas fisik setiap hari.",
      "Ulangi prediksi risiko setelah data kesehatan diperbarui secara konsisten."
    ],
    "disclaimer": "Informasi ini bersifat edukatif dan bukan pengganti konsultasi medis.",
    "source": "rag",
    "fallback_used": false
  }
}
```

Field `data` yang wajib:

- `summary`
- `action_steps`
- `monitoring_plan`
- `disclaimer`
- `source`
- `fallback_used`

Field yang disarankan:

- `goals`
- `safety_notes`

## Aturan Response dari RAG

RAG harus mengikuti aturan berikut:

1. Jawaban harus dalam Bahasa Indonesia.
2. Jangan mengubah angka counterfactual.
3. Jangan membuat target baru yang tidak ada di input.
4. Jangan mengklaim diagnosis.
5. Jangan memberikan instruksi obat atau terapi klinis spesifik.
6. Untuk faktor klinis seperti hipertensi/kolesterol, selalu sarankan pendampingan tenaga kesehatan.
7. Output harus valid JSON.
8. `action_steps` dan `monitoring_plan` tidak boleh kosong.
9. Jika konteks tidak cukup, tetap berikan planner umum berbasis data yang tersedia dan tambahkan safety note.

## Fallback Strategy

Jika RAG gagal, timeout, response invalid, atau response kosong:

1. `diabetify-be` tidak menggagalkan counterfactual result.
2. `diabetify-be` memakai `prescriptive_plan` dari template planner `diabetify-cf`.
3. `diabetify-be` menandai metadata:

```json
{
  "source": "cf_template",
  "fallback_used": true
}
```

Dengan ini mobile tetap mendapat planner walaupun RAG sedang bermasalah.

## Perubahan yang Nanti Dibutuhkan di diabetify-be

Jika endpoint RAG sudah tersedia, `diabetify-be` perlu menambahkan:

```text
internal/rag/client.go
internal/rag/dto.go
```

Config environment:

```env
RAG_SERVICE_URL=http://rag-service:xxxx
RAG_PLANNER_TIMEOUT_MS=5000
RAG_PLANNER_ENABLED=true
```

Integrasi dilakukan saat counterfactual job selesai:

```text
CF result diterima
  -> parse result_payload
  -> jika FEASIBLE, build RAG planner request
  -> call RAG /rag/planner
  -> jika sukses, replace/enrich prescriptive_plan
  -> jika gagal, fallback ke CF template plan
  -> simpan final result_payload
```

Endpoint mobile tidak perlu berubah:

```http
GET /counterfactual/job/{jobId}/result
```

## Perubahan yang Nanti Dibutuhkan di diabetify-cf

Untuk Opsi B, `diabetify-cf` tidak perlu memanggil LLM.

Konfigurasi yang disarankan:

```env
CF_PLANNER_PROVIDER=template
```

atau jika BE sudah sepenuhnya siap fallback:

```env
CF_PLANNER_ENABLED=false
```

Namun untuk transisi aman, lebih baik tetap memakai template planner agar selalu ada fallback plan.

## Perubahan yang Nanti Dibutuhkan di Mobile

Mobile idealnya minimal berubah karena tetap membaca field yang sama:

- `prescriptive_plan.summary`
- `prescriptive_plan.action_steps`
- `prescriptive_plan.monitoring_plan`
- `planner_input.changed_features`

Perubahan opsional:

- tampilkan `safety_notes`
- tampilkan label sumber plan, misalnya `source = rag`
- tampilkan warning jika `fallback_used = true`

## Testing yang Dibutuhkan

### 1. CF feasible + RAG sukses

Expected:

- `result_payload.prescriptive_plan.source = "rag"`
- `fallback_used = false`
- Mobile menampilkan action steps dari RAG.

### 2. CF feasible + RAG timeout

Expected:

- Counterfactual tetap sukses.
- Planner fallback ke template CF.
- `fallback_used = true`.

### 3. CF infeasible

Expected:

- BE tidak perlu memanggil RAG planner.
- Result tetap infeasible.

### 4. RAG response invalid JSON

Expected:

- BE fallback ke template CF.
- Error dicatat di log.

### 5. RAG response kosong

Expected:

- BE fallback ke template CF.

### 6. Mobile save goal

Expected:

- Goal menyimpan final plan hasil RAG atau fallback.
- Planner detail, check-in, completion, dan history tetap berjalan.

## Catatan Penting

Chatbot/RAG tidak perlu menggantikan `diabetify-cf`.

Peran RAG adalah membuat penjelasan dan narasi planner yang lebih grounded, bukan menghitung counterfactual.

Keputusan teknis paling penting:

```text
Mobile hanya bicara ke diabetify-be.
diabetify-be bicara ke diabetify-cf dan chatbot/RAG.
diabetify-cf tidak bicara langsung ke chatbot/RAG.
chatbot/RAG tidak perlu tahu detail UI mobile.
```

Dengan begitu boundary antar modul tetap bersih, API key LLM tetap aman di modul RAG, dan planner tetap punya fallback yang stabil.
