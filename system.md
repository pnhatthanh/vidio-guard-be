# 🛡️ VideoGuard — Kiến Trúc Hệ Thống

> Hệ thống phân tích & kiểm duyệt nội dung video tự động bằng AI

---

## 1. 🗺️ Tổng Quan Kiến Trúc

![Sơ đồ kiến trúc tổng thể VideoGuard](docs/images/architecture_overview.png)

---

## 2. 🧩 Các Thành Phần Công Nghệ

![Technology Icons](docs/images/tech_icons_grid.png)

| Layer | Công nghệ | Vai trò |
|-------|-----------|---------|
| 🖥️ **Frontend** | ![](https://img.shields.io/badge/React-18-61DAFB?logo=react&logoColor=white&style=flat) | Giao diện upload + tracking tiến độ |
| 🌐 **API Server** | ![](https://img.shields.io/badge/Golang-1.22-00ADD8?logo=go&logoColor=white&style=flat) | REST API + WebSocket server |
| ⚡ **Task Queue** | ![](https://img.shields.io/badge/Redis-7-DC382D?logo=redis&logoColor=white&style=flat) ![](https://img.shields.io/badge/Asynq-v0.24-7C3AED?style=flat) | Hàng đợi job bất đồng bộ |
| ⚙️ **Worker** | ![](https://img.shields.io/badge/Golang-Worker-00ADD8?logo=go&logoColor=white&style=flat) | Xử lý video nền |
| 🎬 **Media** | ![](https://img.shields.io/badge/FFmpeg-6.0-007808?logo=ffmpeg&logoColor=white&style=flat) | Tách frame ảnh + âm thanh |
| 🤖 **AI Service** | ![](https://img.shields.io/badge/Python-FastAPI-009688?logo=fastapi&logoColor=white&style=flat) | Inference server AI |
| 🧠 **Image AI** | ![](https://img.shields.io/badge/EfficientNet--B3-TensorFlow-FF6F00?logo=tensorflow&logoColor=white&style=flat) | Phân loại violence/NSFW/safe |
| 🎙️ **Speech AI** | ![](https://img.shields.io/badge/Whisper-OpenAI-412991?logo=openai&logoColor=white&style=flat) | Chuyển âm thanh → văn bản |
| 📝 **Text AI** | ![](https://img.shields.io/badge/PhoBERT-VinAI-0EA5E9?style=flat) | Phân tích văn bản tiếng Việt |
| 🗄️ **Database** | ![](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white&style=flat) | Lưu metadata + kết quả |
| 🪣 **Storage** | ![](https://img.shields.io/badge/MinIO-S3--Compatible-C72E49?logo=minio&logoColor=white&style=flat) | Lưu video, frame, audio |

---

## 3. 🔄 Luồng Xử Lý Bất Đồng Bộ

![Luồng xử lý dữ liệu](docs/images/data_flow.png)

### Giải thích từng bước:

```
👤 User                  🌐 Golang API            ⚡ Redis Queue
   │                          │                         │
   │── POST /upload ─────────▶│                         │
   │                          │── PUT video ──▶ 🪣 MinIO│
   │                          │── INSERT ──────▶ 🗄️ PG  │
   │                          │── ENQUEUE ─────────────▶│
   │◀── 202 Accepted ─────────│                         │
   │                          │                         │
   │                    ⚙️ Worker ◀── DEQUEUE ──────────│
   │                          │
   │         ┌────────────────┴────────────────┐
   │         │  Parallel goroutines              │
   │         ▼                                  ▼
   │   🎬 FFmpeg                          🎬 FFmpeg
   │   Extract Frames                     Extract Audio
   │         │                                  │
   │         ▼                                  ▼
   │   🧠 EfficientNet                   🎙️ Whisper ASR
   │   (Violence/NSFW/Safe)              (Audio → Text)
   │         │                                  │
   │         │                           📝 PhoBERT
   │         │                           (Text classify)
   │         └────────────────┬────────────────┘
   │                          ▼
   │                   📊 Result Aggregator
   │                          │
   │                          ├──▶ 🗄️ PostgreSQL
   │                          └──▶ ⚡ Redis Pub/Sub
   │                                      │
   │◀── WebSocket Progress ───────────────┘
```

---

## 4. 🤖 Pipeline Xử Lý AI

![AI Processing Pipeline](docs/images/ai_pipeline.png)

### 4.1 🎥 Video Track — EfficientNet-B3

```mermaid
flowchart LR
    A["🎬\nVideo\nMP4"] -->|"ffmpeg\n-r 1fps"| B["🖼️\nFrames\nJPG"]
    B -->|"batch\n32 frames"| C["🧠\nEfficientNet-B3\nTF/Keras"]
    C --> D["📊\nSoftmax\n3 classes"]
    D --> E1["🔴 Violence\n0.82"]
    D --> E2["🟣 NSFW\n0.11"]
    D --> E3["🟢 Safe\n0.07"]

    style A fill:#1e3a5f,color:#fff,stroke:#3b82f6
    style B fill:#1e3a5f,color:#fff,stroke:#3b82f6
    style C fill:#7c2d12,color:#fff,stroke:#f97316
    style D fill:#581c87,color:#fff,stroke:#a855f7
    style E1 fill:#7f1d1d,color:#fff,stroke:#ef4444
    style E2 fill:#4a1d96,color:#fff,stroke:#8b5cf6
    style E3 fill:#14532d,color:#fff,stroke:#22c55e
```


### 4.2 🎙️ Audio Track — Whisper + PhoBERT

```mermaid
flowchart LR
    A["🎬\nVideo\nMP4"] -->|"ffmpeg -vn\n-ar 16000"| B["🔊\naudio.wav\n16kHz Mono"]
    B --> C["🎙️\nWhisper\nopenai/whisper-medium"]
    C -->|"transcript\ntiếng Việt"| D["📝\nPhoBERT\nvinai/phobert-base-v2"]
    D --> E["🏷️\nClassification\nHead"]
    E --> F1["✅ clean"]
    E --> F2["❌ offensive"]
    E --> F3["❌ hate"]

    style A fill:#1e3a5f,color:#fff,stroke:#3b82f6
    style B fill:#1e3a5f,color:#fff,stroke:#3b82f6
    style C fill:#1e1b4b,color:#fff,stroke:#818cf8
    style D fill:#064e3b,color:#fff,stroke:#10b981
    style E fill:#4a1d96,color:#fff,stroke:#a855f7
    style F1 fill:#14532d,color:#fff,stroke:#22c55e
    style F2 fill:#7f1d1d,color:#fff,stroke:#ef4444
    style F3 fill:#7f1d1d,color:#fff,stroke:#ef4444
    style F4 fill:#7f1d1d,color:#fff,stroke:#ef4444
```

---

### 4.3 📊 Tổng Hợp Kết Quả — Result Aggregator

```mermaid
flowchart TD
    A["🖼️ Frame Results\n{violence, nsfw, safe}\nper frame"] --> C
    B["🔊 Audio Result\n{violation_type,\nconfidence}"] --> C

    C["📊 Result Aggregator\nGolang Worker"] --> D["🧮 Compute\nRisk Score"]

    D --> E{"risk_score >= ?"}

    E -->|">= 0.75\nhoặc peak >= 0.9"| F["🔴 REJECTED"]
    E -->|">= 0.45"| G["🟡 REVIEW\nREQUIRED"]
    E -->|"< 0.45"| H["🟢 APPROVED"]

    F --> I[("🗄️ PostgreSQL\nSave verdict")]
    G --> I
    H --> I
    I --> J["⚡ Redis Pub/Sub\nNotify DONE"]
    J --> K["🌐 WebSocket\nNotify Frontend"]

    style C fill:#4a1d96,color:#fff,stroke:#a855f7
    style D fill:#1e3a5f,color:#fff,stroke:#3b82f6
    style F fill:#7f1d1d,color:#fff,stroke:#ef4444
    style G fill:#78350f,color:#fff,stroke:#f59e0b
    style H fill:#14532d,color:#fff,stroke:#22c55e
    style I fill:#1e3a5f,color:#fff,stroke:#6366f1
    style J fill:#7f1d1d,color:#fff,stroke:#dc2626
    style K fill:#0c4a6e,color:#fff,stroke:#0284c7
```

<!-- **Công thức tính Risk Score:**

```
risk_score = 0.35 × avg_violence
           + 0.25 × avg_nsfw
           + 0.25 × audio_confidence × (1.5 nếu vi phạm, 1.0 nếu không)
           + 0.15 × peak_violence
```

--- -->

## 5. 🗄️ Cơ Sở Dữ Liệu

```mermaid
erDiagram
    USERS {
        uuid id PK
        varchar email UK
        varchar password
        varchar full_name
        varchar avatar_url
        varchar role
        timestamp created_at
    }

    VIDEOS {
        uuid id PK
        uuid user_id FK
        varchar original_filename
        varchar video_url
        bigint file_size_bytes
        int duration_seconds
        varchar status
        int progress_percent
        varchar current_stage
        timestamp uploaded_at
        timestamp processed_at
    }

    FRAME_RESULTS {
        uuid id PK
        uuid video_id FK
        int frame_number
        varchar frame_url
        float8 timestamp_seconds
        float8 violence_score
        float8 nsfw_score
        float8 safe_score
        varchar predicted_label
    }

    AUDIO_RESULTS {
        uuid id PK
        uuid video_id FK
        text transcript
        float8 offensive_score
        float8 hate_score
        float8 clean_score
        varchar predicted_label
    }

    FINAL_VERDICTS {
        uuid id PK
        uuid video_id FK
        varchar verdict
        float8 risk_score
        float8 peak_violence_score
        float8 peak_nsfw_score
        int flagged_frames_count
        jsonb flagged_timestamps
    }

    USERS ||--o{ VIDEOS : "uploads"
    VIDEOS ||--o{ FRAME_RESULTS : "has frames"
    VIDEOS ||--o| AUDIO_RESULTS : "has audio"
    VIDEOS ||--o| FINAL_VERDICTS : "receives verdict"
```

---

## 6. 📡 WebSocket — Tracking Tiến Độ Thời Gian Thực

```mermaid
sequenceDiagram
    participant 👤 as 👤 User (Browser)
    participant ⚛️ as ⚛️ ReactJS FE
    participant 🌐 as 🌐 Golang API
    participant ⚡ as ⚡ Redis Pub/Sub
    participant ⚙️ as ⚙️ Golang Worker

    👤->>⚛️: Upload video
    ⚛️->>🌐: POST /api/v1/videos/upload
    🌐-->>⚛️: 202 + {video_id, ws_url}
    ⚛️->>🌐: WS Connect /ws/videos/{id}

    Note over ⚛️,🌐: WebSocket kết nối

    ⚙️->>⚡: PUBLISH progress {0%, "starting"}
    ⚡->>🌐: Forward event
    🌐-->>⚛️: {progress: 0, stage: "starting"}
    ⚛️-->>👤: Progress bar 0%

    ⚙️->>⚡: PUBLISH progress {15%, "frame_extraction"}
    ⚡->>🌐: Forward
    🌐-->>⚛️: {progress: 15, stage: "frame_extraction"}
    ⚛️-->>👤: Progress bar 15%

    ⚙️->>⚡: PUBLISH progress {50%, "video_analysis"}
    🌐-->>⚛️: {progress: 50}
    ⚛️-->>👤: Progress bar 50%

    ⚙️->>⚡: PUBLISH progress {75%, "audio_analysis"}
    🌐-->>⚛️: {progress: 75}

    ⚙️->>⚡: PUBLISH {100%, "COMPLETED"}
    🌐-->>⚛️: {status: "COMPLETED", progress: 100}
    ⚛️-->>👤: ✅ Hiển thị kết quả
```

### Progress Stages

| % | Stage | Mô tả |
|---|-------|-------|
| 0% | `starting` | Worker bắt đầu xử lý |
| 15% | `frame_extraction` | FFmpeg đang tách frame |
| 20–50% | `audio_extraction` | FFmpeg tách audio |
| 50% |  `video_analysis` | EfficientNet phân tích từng frame |
| 50–75% | `audio_analysis` | Whisper + PhoBERT xử lý |
| 90% | `aggregation` | Tổng hợp kết quả |
| 100% | `completed` | Hoàn tất, lưu DB |

---

## 7. 🗂️ Cấu Trúc Lưu Trữ MinIO

```
🪣 Bucket: videoguard
│
└── 📁 videos/
    └── 📁 {video_id}/               ← UUID mỗi video
        ├── 🎬 original.mp4           ← File gốc người dùng upload
        ├── 🔊 audio.wav              ← Audio tách ra bởi FFmpeg
        └── 📁 frames/
            ├── 🖼️ frame_0001.jpg    ← Frame tại t=1s
            ├── 🖼️ frame_0002.jpg    ← Frame tại t=2s
            ├── 🖼️ frame_0003.jpg    ← Frame tại t=3s
            └── ...
```

**Lifecycle Policies:**
- 🔒 Bucket **không public** — truy cập qua pre-signed URL (TTL 10 phút)
- 🗑️ Frame images tự động xóa sau **7 ngày**
- 💾 Original video giữ lại theo policy người dùng

---

## 8. 🚀 Deployment (Docker Compose)

```mermaid
graph TB
    subgraph NET["🌐 Internet"]
        U["👤 Users"]
    end

    subgraph PROXY["🔀 Nginx — Reverse Proxy"]
        N["nginx:1.25\n:443 HTTPS\nSSL + Load Balance"]
    end

    subgraph APPS["📦 App Services"]
        FE["⚛️ videoguard-fe\nReactJS :3000"]
        API["🌐 videoguard-api\nGolang API :8080"]
        W["⚙️ videoguard-worker\n× 2 replicas"]
        AI["🤖 videoguard-ai\nPython FastAPI :8000\n(+ GPU optional)"]
    end

    subgraph DATA["💾 Data Services"]
        PG["🐘 postgres:16\n:5432"]
        RD["🔴 redis:7\n:6379"]
        MN["🪣 minio\n:9000 API\n:9001 Console"]
    end

    U -->|HTTPS| N
    N -->|"/"| FE
    N -->|"/api/*\n/ws/*"| API
    API <--> PG & RD & MN
    W <--> PG & RD & MN
    W -->|HTTP| AI

    style NET fill:#0d1117,stroke:#e94560,color:#fff
    style PROXY fill:#161b22,stroke:#0f3460,color:#fff
    style APPS fill:#0f3460,stroke:#533483,color:#fff
    style DATA fill:#533483,stroke:#e94560,color:#fff
```

---

## 9. 🔌 API Reference

### REST Endpoints

| Method | Endpoint | Mô tả |
|--------|----------|-------|
| `POST` | `/api/v1/videos/upload` | Upload video mới |
| `GET` | `/api/v1/videos` | Danh sách video của user |
| `GET` | `/api/v1/videos/:id` | Chi tiết + trạng thái |
| `GET` | `/api/v1/videos/:id/result` | Báo cáo kiểm duyệt đầy đủ |
| `GET` | `/api/v1/videos/:id/frames` | Danh sách frame results |
| `DELETE` | `/api/v1/videos/:id` | Xóa video |
| `WS` | `/ws/videos/:id` | WebSocket real-time progress |
| `GET` | `/api/v1/health` | Health check |

<!-- ### WebSocket Message Format

```json
{
  "event": "progress_update",
  "video_id": "550e8400-...",
  "status": "PROCESSING",
  "progress": 50,
  "current_stage": "video_analysis",
  "stages": {
    "upload":           "completed",
    "frame_extraction": "completed",
    "video_analysis":   "in_progress",
    "audio_extraction": "pending",
    "audio_analysis":   "pending",
    "aggregation":      "pending"
  }
}
```

### Final Result Response

```json
{
  "video_id": "550e8400-...",
  "verdict": "REJECTED",
  "risk_score": 0.87,
  "summary": {
    "total_frames": 120,
    "flagged_frames": 23,
    "peak_violence": 0.94,
    "audio_violation": true,
    "audio_type": "threat",
    "flagged_timestamps": [1.5, 3.2, 7.8, 12.1]
  }
}
```

--- -->

> 📅 VideoGuard System Architecture v1.0 — 2026-04-14
