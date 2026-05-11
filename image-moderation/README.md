# 🤖 VideoGuard AI Service

> **FastAPI** inference server cho mô hình **EfficientNet** — phân loại frame video thành `violence` / `nsfw` / `safe`.

---

## 📁 Cấu trúc thư mục

```
ai-service/
├── app/
│   ├── main.py          # FastAPI app factory + lifespan
│   ├── config.py        # Settings via pydantic-settings (prefix: AI_)
│   ├── model.py         # Load model, preprocess, batch predict
│   ├── schemas.py       # Pydantic request/response models
│   └── routers/
│       ├── predict.py   # POST /predict/batch
│       └── health.py    # GET  /health
├── models/              # Đặt file efficientnet.keras vào đây
├── tests/
│   └── test_predict.py  # Integration tests (no real model needed)
├── .env                 # Local env config
├── .env.example         # Template
├── requirements.txt
├── Dockerfile
└── run.py               # Uvicorn entry point
```

---

## 🚀 Chạy local

### 1. Cài dependencies

```bash
cd ai-service
python -m venv .venv
.venv\Scripts\activate        # Windows
# source .venv/bin/activate   # Linux/Mac

pip install -r requirements.txt
```

### 2. Đặt model vào thư mục `models/`

```
ai-service/
└── models/
    └── efficientnet.keras     ← file model train được
```

### 3. Chạy service

```bash
python run.py
# hoặc
uvicorn app.main:app --host 0.0.0.0 --port 8000 --reload
```

Service sẽ chạy tại: **http://localhost:8000**

---

## 📡 API Endpoints

### `GET /health`

Kiểm tra trạng thái service.

```json
{
  "status": "ok",
  "model_loaded": true,
  "labels": ["nsfw", "safe", "violence"]
}
```

---

### `POST /predict/batch`

Nhận nhiều frame dưới dạng `multipart/form-data`, trả về dự đoán cho từng frame.

**Request** — `Content-Type: multipart/form-data`

| Field  | Type            | Mô tả                     |
|--------|-----------------|---------------------------|
| `files`| `UploadFile[]`  | Danh sách frame JPEG/PNG  |

**Response**

```json
{
  "total": 3,
  "predictions": [
    {
      "frame": "frame_0001.jpg",
      "label": "safe",
      "confidence": 0.97,
      "scores": {
        "nsfw":     0.01,
        "safe":     0.97,
        "violence": 0.02
      }
    },
    {
      "frame": "frame_0002.jpg",
      "label": "violence",
      "confidence": 0.89,
      "scores": {
        "nsfw":     0.03,
        "safe":     0.08,
        "violence": 0.89
      }
    }
  ]
}
```

**Giới hạn:** Tối đa `AI_BATCH_SIZE` frames / request (mặc định: 32).

---

## 🐳 Docker

### Build & chạy riêng lẻ

```bash
docker build -t videoguard-ai .
docker run -p 8000:8000 \
  -v $(pwd)/models:/app/models \
  --env-file .env \
  videoguard-ai
```

### Chạy cùng hệ thống (docker-compose)

```bash
# Từ thư mục gốc vidio-guard-be/
docker-compose up videoguard-ai
```

> **GPU:** Uncomment phần `deploy.resources` trong `docker-compose.yml` nếu host có NVIDIA GPU + `nvidia-container-toolkit`.

---

## ⚙️ Cấu hình Environment

Tất cả biến cấu hình đều có prefix `AI_`:

| Biến               | Mặc định                     | Mô tả                        |
|--------------------|------------------------------|------------------------------|
| `AI_MODEL_PATH`    | `models/efficientnet.keras`  | Đường dẫn đến file model     |
| `AI_IMG_SIZE`      | `224`                        | Kích thước ảnh đầu vào       |
| `AI_BATCH_SIZE`    | `32`                         | Số frame tối đa / request    |
| `AI_LABELS`        | `["nsfw","safe","violence"]` | Thứ tự nhãn output của model |
| `AI_HOST`          | `0.0.0.0`                    | Bind host                    |
| `AI_PORT`          | `8000`                       | Bind port                    |
| `AI_WORKERS`       | `1`                          | Số Uvicorn workers (giữ = 1 nếu dùng GPU) |
| `AI_LOG_LEVEL`     | `info`                       | Log level                    |

---

## 🧪 Chạy tests

```bash
pytest tests/ -v
```

Tests dùng **monkeypatch** — không cần file model thật.

---

## 🔗 Golang Worker gọi AI Service

```go
func sendBatchToAI(ctx context.Context, frames [][]byte, aiURL string) ([]FramePrediction, error) {
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)

    for i, frame := range frames {
        part, _ := writer.CreateFormFile("files", fmt.Sprintf("frame_%04d.jpg", i))
        part.Write(frame)
    }
    writer.Close()

    req, _ := http.NewRequestWithContext(ctx, "POST", aiURL+"/predict/batch", body)
    req.Header.Set("Content-Type", writer.FormDataContentType())

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result BatchPredictResponse
    json.NewDecoder(resp.Body).Decode(&result)
    return result.Predictions, nil
}
```

---

## 📖 Swagger UI

Khi service đang chạy, truy cập:
- **Swagger**: http://localhost:8000/docs
- **ReDoc**: http://localhost:8000/redoc
