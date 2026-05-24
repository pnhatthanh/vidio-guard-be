# Vidio Guard — API Reference

Tài liệu tham chiếu REST API và WebSocket của backend Go (`video-api`).

| Thuộc tính | Giá trị |
|------------|---------|
| **Base URL** | `http://localhost:8080` (mặc định, cấu hình qua `HTTP_ADDR`) |
| **API prefix** | `/api/v1` |
| **Content-Type** | `application/json` (trừ upload dùng `multipart/form-data`) |
| **CORS** | `Access-Control-Allow-Origin: *` |

---

## Mục lục

1. [Xác thực (Authentication)](#1-xác-thực-authentication)
2. [Định dạng lỗi chung](#2-định-dạng-lỗi-chung)
3. [Auth API — `/api/v1/auth`](#3-auth-api--apiv1auth)
4. [User API — `/api/v1/users`](#4-user-api--apiv1users)
5. [Video API — `/api/v1/videos`](#5-video-api--apiv1videos)
6. [WebSocket — `/api/v1/ws/pipeline`](#6-websocket--apiv1wspipeline)
7. [Tham chiếu enum & giá trị](#7-tham-chiếu-enum--giá-trị)

---

## 1. Xác thực (Authentication)

### 1.1 Access token (JWT)

Các endpoint được đánh dấu **Yêu cầu JWT** cần header:

```http
Authorization: Bearer <access_token>
```

| Thuộc tính | Mặc định | Env |
|------------|---------|-----|
| Thời hạn access token | 15 phút | `JWT_ACCESS_TTL` |
| Thời hạn refresh token | 168 giờ (7 ngày) | `JWT_REFRESH_TTL` |

Sau khi logout, access token bị đưa vào blacklist (Redis) cho đến khi hết hạn.

### 1.2 Endpoint không cần JWT

| Method | Path |
|--------|------|
| `POST` | `/api/v1/auth/register` |
| `POST` | `/api/v1/auth/login` |
| `POST` | `/api/v1/auth/google` |
| `POST` | `/api/v1/auth/refresh` |
| `GET` | `/api/v1/ws/pipeline` (auth qua query/header, xem mục 6) |

### 1.3 Endpoint cần JWT

| Method | Path |
|--------|------|
| `POST` | `/api/v1/auth/logout` |
| `GET` | `/api/v1/users/me` |
| `PATCH` | `/api/v1/users/me` |
| `PATCH` | `/api/v1/users/me/password` |
| `GET` | `/api/v1/videos` |
| `POST` | `/api/v1/videos/upload` |
| `GET` | `/api/v1/videos/:id/status` |

---

## 2. Định dạng lỗi chung

Mọi lỗi nghiệp vụ trả về JSON cùng cấu trúc:

```json
{
  "code": 400,
  "type": "bad_request",
  "message": "mô tả lỗi cụ thể"
}
```

### 2.1 Bảng HTTP status ↔ `type`

| HTTP | `type` | Ý nghĩa |
|------|--------|---------|
| `400` | `bad_request` | Request sai format, thiếu field, validation fail |
| `401` | `unauthorized` | Chưa đăng nhập, token sai/hết hạn/bị revoke |
| `403` | `forbidden` | Không có quyền (hiện chưa dùng nhiều) |
| `404` | `not_found` | Resource không tồn tại |
| `409` | `conflict` | Trùng dữ liệu (vd: email đã đăng ký) |
| `500` | `internal_error` | Lỗi server nội bộ |

### 2.2 Ví dụ lỗi validation (400)

Khi body JSON không đúng rule `binding` của Gin validator:

```json
{
  "code": 400,
  "type": "bad_request",
  "message": "Key: 'RegisterRequest.Email' Error:Field validation for 'Email' failed on the 'email' tag"
}
```

### 2.3 Ví dụ lỗi unauthorized (401)

```json
{
  "code": 401,
  "type": "unauthorized",
  "message": "missing bearer token"
}
```

```json
{
  "code": 401,
  "type": "unauthorized",
  "message": "invalid token"
}
```

```json
{
  "code": 401,
  "type": "unauthorized",
  "message": "token revoked"
}
```

### 2.4 Ví dụ lỗi not found (404)

```json
{
  "code": 404,
  "type": "not_found",
  "message": "video not found"
}
```

### 2.5 Ví dụ lỗi conflict (409)

```json
{
  "code": 409,
  "type": "conflict",
  "message": "email already registered"
}
```

### 2.6 Lỗi không xác định (500)

Nếu handler trả error không phải `AppError`, middleware trả:

```json
{
  "code": 500,
  "type": "internal_error",
  "message": "internal server error"
}
```

---

## 3. Auth API — `/api/v1/auth`

### 3.1 `POST /auth/register`

Đăng ký tài khoản email/password. **Không trả token** — client cần gọi `/auth/login` sau khi đăng ký.

**Auth:** Không cần

**Request body:**

```json
{
  "full_name": "Nguyen Van A",
  "email": "user@example.com",
  "password": "password123",
  "confirm_password": "password123"
}
```

| Field | Type | Bắt buộc | Rule |
|-------|------|----------|------|
| `full_name` | string | ✅ | `min=2`, `max=100` |
| `email` | string | ✅ | định dạng email hợp lệ |
| `password` | string | ✅ | `min=8` |
| `confirm_password` | string | ✅ | phải bằng `password` |

**Response `201 Created`:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "full_name": "Nguyen Van A",
  "email": "user@example.com"
}
```

| Field | Type | Mô tả |
|-------|------|-------|
| `id` | UUID string | ID user |
| `full_name` | string | Họ tên |
| `email` | string | Email (lowercase, trimmed) |

**Lỗi thường gặp:**

| HTTP | `message` |
|------|-----------|
| `400` | Validation message từ Gin (thiếu field, password < 8, confirm không khớp…) |
| `409` | `email already registered` |
| `500` | `failed to check email`, `failed to hash password`, `failed to create user` |

---

### 3.2 `POST /auth/login`

Đăng nhập bằng email/password.

**Auth:** Không cần

**Request body:**

```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

| Field | Type | Bắt buộc | Rule |
|-------|------|----------|------|
| `email` | string | ✅ | email hợp lệ |
| `password` | string | ✅ | không rỗng |

**Response `200 OK`:**

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "opaque-random-string"
}
```

| Field | Type | Mô tả |
|-------|------|-------|
| `access_token` | string | JWT dùng cho các API protected |
| `refresh_token` | string | Opaque token, lưu DB (hash SHA-256) |

**Lỗi thường gặp:**

| HTTP | `message` |
|------|-----------|
| `400` | Validation message |
| `401` | `invalid credentials` (email không tồn tại, sai password, hoặc tài khoản chỉ Google) |
| `500` | `failed to find user`, `failed to generate tokens`, `failed to store refresh token` |

---

### 3.3 `POST /auth/google`

Đăng nhập / đăng ký bằng Google ID token.

**Auth:** Không cần

**Request body:**

```json
{
  "id_token": "eyJhbGciOiJSUzI1NiIs..."
}
```

| Field | Type | Bắt buộc | Rule |
|-------|------|----------|------|
| `id_token` | string | ✅ | Google ID token từ client SDK |

**Logic server:**
- Verify token qua `https://oauth2.googleapis.com/tokeninfo`
- Nếu `GOOGLE_CLIENT_ID` được cấu hình, kiểm tra `aud` khớp
- User đã có Google ID → login
- Email đã tồn tại (đăng ký password) → link Google ID
- Chưa có → tạo user mới

**Response `200 OK`:** Giống login

```json
{
  "access_token": "eyJhbG...",
  "refresh_token": "..."
}
```

**Lỗi thường gặp:**

| HTTP | `message` |
|------|-----------|
| `400` | Validation message (`id_token` required) |
| `401` | `invalid Google token` |
| `500` | `failed to find user`, `failed to link Google account`, … |

---

### 3.4 `POST /auth/refresh`

Đổi refresh token lấy cặp token mới. Refresh token cũ bị **xóa** (rotation).

**Auth:** Không cần

**Request body:**

```json
{
  "refresh_token": "opaque-refresh-token"
}
```

| Field | Type | Bắt buộc |
|-------|------|----------|
| `refresh_token` | string | ✅ |

**Response `200 OK`:**

```json
{
  "access_token": "eyJhbG...",
  "refresh_token": "new-opaque-token"
}
```

**Lỗi thường gặp:**

| HTTP | `message` |
|------|-----------|
| `400` | Validation message |
| `401` | `invalid refresh token`, `refresh token expired` |
| `500` | `failed to validate refresh token`, `failed to generate tokens` |

---

### 3.5 `POST /auth/logout`

Thu hồi access token hiện tại (blacklist) và xóa refresh token.

**Auth:** ✅ JWT bắt buộc

**Request body:**

```json
{
  "refresh_token": "opaque-refresh-token"
}
```

| Field | Type | Bắt buộc |
|-------|------|----------|
| `refresh_token` | string | ✅ |

**Response `200 OK`:**

```json
{
  "message": "logged out"
}
```

**Lỗi thường gặp:**

| HTTP | `message` |
|------|-----------|
| `400` | Validation message, `invalid session`, `refresh_token is required` |
| `401` | `unauthorized`, `missing bearer token`, `invalid token`, `token revoked`, `invalid session` |
| `500` | `failed to revoke token`, `failed to revoke refresh token` |

> **Lưu ý:** Nếu refresh token không tồn tại trong DB, logout vẫn thành công (access token vẫn bị blacklist).

---

## 4. User API — `/api/v1/users`

Tất cả endpoint trong group này **yêu cầu JWT**.

### 4.1 `GET /users/me`

Lấy profile user đang đăng nhập.

**Auth:** ✅ JWT

**Request:** Không có body/query.

**Response `200 OK`:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "full_name": "Nguyen Van A",
  "email": "user@example.com",
  "avatar_url": "https://cdn.example.com/avatar.jpg",
  "has_password": true,
  "has_google": false,
  "created_at": "2026-01-15T08:00:00Z"
}
```

| Field | Type | Mô tả |
|-------|------|-------|
| `id` | string | UUID user |
| `full_name` | string | Họ tên |
| `email` | string | Email (read-only qua API) |
| `avatar_url` | string | URL avatar; `""` nếu chưa set |
| `has_password` | boolean | `true` nếu đăng ký email/password |
| `has_google` | boolean | `true` nếu đã link Google |
| `created_at` | ISO 8601 | Thời điểm tạo tài khoản |

**Lỗi thường gặp:**

| HTTP | `message` |
|------|-----------|
| `401` | `missing bearer token`, `invalid token`, `authentication required` |
| `404` | `user not found` |
| `500` | `failed to load user` |

---

### 4.2 `PATCH /users/me`

Cập nhật `full_name` và/hoặc `avatar_url`. **Email không thể đổi qua API.**

**Auth:** ✅ JWT

**Request body:**

```json
{
  "full_name": "Nguyen Van B",
  "avatar_url": "https://cdn.example.com/new-avatar.jpg"
}
```

| Field | Type | Bắt buộc | Rule |
|-------|------|----------|------|
| `full_name` | string | ✅ | `min=2`, `max=100` |
| `avatar_url` | string \| null | ❌ | `max=500`; gửi `""` hoặc `null` để xóa avatar |

**Response `200 OK`:** Cùng format `GET /users/me`.

**Lỗi thường gặp:**

| HTTP | `message` |
|------|-----------|
| `400` | Validation message |
| `401` | Token errors |
| `404` | `user not found` |
| `500` | `failed to update profile` |

---

### 4.3 `PATCH /users/me/password`

Đổi mật khẩu (chỉ tài khoản có password).

**Auth:** ✅ JWT

**Request body:**

```json
{
  "current_password": "oldpass123",
  "new_password": "newpass123",
  "confirm_new_password": "newpass123"
}
```

| Field | Type | Bắt buộc | Rule |
|-------|------|----------|------|
| `current_password` | string | ✅ | — |
| `new_password` | string | ✅ | `min=8` |
| `confirm_new_password` | string | ✅ | phải bằng `new_password` |

**Response `200 OK`:**

```json
{
  "message": "password updated"
}
```

**Lỗi thường gặp:**

| HTTP | `message` |
|------|-----------|
| `400` | Validation message, `account uses Google sign-in; password cannot be changed` |
| `401` | `current password is incorrect`, token errors |
| `404` | `user not found` |
| `500` | `failed to hash password`, `failed to update password` |

---

## 5. Video API — `/api/v1/videos`

Tất cả endpoint trong group này **yêu cầu JWT**. User chỉ truy cập video của chính mình.

### 5.1 `GET /videos`

Danh sách video của user hiện tại — dùng cho dashboard (tìm kiếm, lọc, sort, phân trang).

**Auth:** ✅ JWT

**Query parameters:**

| Param | Mặc định | Giá trị hợp lệ | Mô tả |
|-------|----------|----------------|-------|
| `q` | — | string | Tìm theo tên file (`ILIKE`) |
| `status` | `all` | `all`, `uploaded`, `processing`, `completed`, `failed` | Lọc trạng thái xử lý |
| `filter` | `all` | `all`, `violated`, `safe` | Lọc kết quả moderation (video `completed`) |
| `days` | `0` | `0`, `7`, `30`, … | `0` = all time; N = rolling window N ngày gần đây |
| `sort` | `processed_at` | `processed_at`, `uploaded_at`, `risk_score`, `filename` | Trường sort |
| `order` | `desc` | `asc`, `desc` | Thứ tự |
| `page` | `1` | ≥ 1 | Trang (tự normalize nếu < 1) |
| `limit` | `20` | 1–100 | Số item/trang (max 100) |

**Ví dụ request:**

```http
GET /api/v1/videos?days=7&filter=violated&sort=processed_at&order=desc&page=1&limit=20
Authorization: Bearer <access_token>
```

**Response `200 OK`:**

```json
{
  "items": [
    {
      "video_id": "550e8400-e29b-41d4-a716-446655440000",
      "video_url": "http://localhost:9000/videos/550e8400-e29b-41d4-a716-446655440000.mp4?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
      "original_filename": "Security_Cam_04.mp4",
      "status": "completed",
      "stage": "completed",
      "progress_percent": 100,
      "file_size_bytes": 188743680,
      "uploaded_at": "2026-05-20T08:00:00Z",
      "processed_at": "2026-05-20T08:05:00Z",
      "video_url": "http://localhost:9000/videos/550e8400-e29b-41d4-a716-446655440000.mp4?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
      "verdict": "violation",
      "violated": true,
      "risk_score": 0.72,
      "violation_count": 3
    }
  ],
  "total": 42,
  "page": 1,
  "limit": 20,
  "total_pages": 3
}
```

**Response fields — root:**

| Field | Type | Mô tả |
|-------|------|-------|
| `items` | array | Danh sách video |
| `total` | int64 | Tổng số bản ghi khớp filter |
| `page` | int | Trang hiện tại |
| `limit` | int | Limit sau normalize |
| `total_pages` | int | `ceil(total / limit)` |

**Response fields — `items[]`:**

| Field | Type | Mô tả |
|-------|------|-------|
| `video_id` | string | UUID video |
| `video_url` | string | Presigned GET URL MinIO (host `MINIO_PUBLIC_ENDPOINT`); dùng trực tiếp `<video src>` |
| `original_filename` | string | Tên file gốc khi upload |
| `status` | string | `uploaded` \| `processing` \| `completed` \| `failed` |
| `stage` | string | Stage hiện tại (xem mục 7) |
| `progress_percent` | int | 0–100 |
| `file_size_bytes` | int64 | Kích thước file |
| `uploaded_at` | ISO 8601 | Thời điểm upload |
| `processed_at` | ISO 8601 \| null | Thời điểm xử lý xong/thất bại |
| `verdict` | string | `safe` \| `warning` \| `violation` (chỉ khi có verdict) |
| `violated` | boolean | `true` nếu `verdict != "safe"` |
| `risk_score` | float | 0.0–1.0 (chỉ khi có verdict) |
| `violation_count` | int | Số đoạn vi phạm trong `violation_segments` |

**Lỗi thường gặp:**

| HTTP | `message` |
|------|-----------|
| `400` | `sort must be one of: processed_at, uploaded_at, risk_score, filename` |
| `400` | `order must be asc or desc` |
| `400` | `status must be one of: all, uploaded, processing, completed, failed` |
| `400` | `filter must be one of: all, violated, safe` |
| `401` | Token errors |
| `500` | `failed to list videos` |

---

### 5.2 `POST /videos/upload`

Upload video và enqueue xử lý nền.

**Auth:** ✅ JWT

**Content-Type:** `multipart/form-data`

**Form fields:**

| Field | Type | Bắt buộc | Mô tả |
|-------|------|----------|------|
| `file` | file | ✅ | File video |

**Ví dụ request (curl):**

```bash
curl -X POST "http://localhost:8080/api/v1/videos/upload" \
  -H "Authorization: Bearer <access_token>" \
  -F "file=@/path/to/video.mp4"
```

**Luồng server:**
1. Tạo record `videos` (`status=uploaded`)
2. Upload lên MinIO (`objectKey = {video_id}{ext}`)
3. Cập nhật `status=processing`, `stage=starting`
4. Enqueue job Asynq

**Response `201 Created`:**

```json
{
  "video_id": "550e8400-e29b-41d4-a716-446655440000",
  "video_url": "http://localhost:9000/videos/550e8400-e29b-41d4-a716-446655440000.mp4?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
  "status": "processing",
  "stage": "starting",
  "progress_percent": 0
}
```

| Field | Mô tả |
|-------|--------|
| `video_url` | Presigned URL MinIO (TTL `MINIO_PRESIGN_TTL`); `<video src={video_url} />` |

| Field | Type | Mô tả |
|-------|------|-------|
| `video_id` | string | UUID video mới |
| `status` | string | Luôn `processing` sau upload thành công |
| `stage` | string | `starting` |
| `progress_percent` | int | `0` |

**Lỗi thường gặp:**

| HTTP | `message` |
|------|-----------|
| `400` | `file is required` |
| `400` | `file size must be greater than 0` |
| `401` | Token errors |
| `500` | `could not read file`, `failed to create video record`, `failed to store video`, `failed to update video status`, `failed to enqueue video processing` |

> Upload fail sau khi tạo DB record → video được mark `failed`.

---

### 5.3 `GET /videos/:id/status`

Tra cứu trạng thái xử lý và kết quả moderation của một video.

**Auth:** ✅ JWT (chỉ video thuộc user hiện tại)

**Path parameters:**

| Param | Type | Mô tả |
|-------|------|-------|
| `id` | UUID | ID video |

**Ví dụ request:**

```http
GET /api/v1/videos/550e8400-e29b-41d4-a716-446655440000/status
Authorization: Bearer <access_token>
```

#### Response `200 OK` — đang xử lý

```json
{
  "video_id": "550e8400-e29b-41d4-a716-446655440000",
  "video_url": "http://localhost:9000/videos/550e8400-e29b-41d4-a716-446655440000.mp4?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
  "status": "processing",
  "stage": "frame_analysis",
  "progress_percent": 50,
  "original_filename": "clip.mp4",
  "uploaded_at": "2026-05-23T10:00:00Z",
  "processed_at": null
}
```

#### Response `200 OK` — hoàn tất (có vi phạm)

```json
{
  "video_id": "550e8400-e29b-41d4-a716-446655440000",
  "video_url": "http://localhost:9000/videos/550e8400-e29b-41d4-a716-446655440000.mp4?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
  "status": "completed",
  "stage": "completed",
  "progress_percent": 100,
  "original_filename": "clip.mp4",
  "uploaded_at": "2026-05-23T10:00:00Z",
  "processed_at": "2026-05-23T10:05:30Z",
  "verdict": {
    "verdict": "violation",
    "violated": true,
    "risk_score": 0.72,
    "final_score": 0.72,
    "frame_score": 0.65,
    "audio_score": 0.88,
    "total_frames": 113,
    "video_duration_sec": 120.5,
    "hard_rule_triggered": false,
    "hard_rule_reason": "",
    "transcript": "Xin chào mọi người. Nội dung không phù hợp..."
  },
  "violation_segments": [
    {
      "source": "visual",
      "category": "violence",
      "start_sec": 4.0,
      "end_sec": 7.0,
      "peak_score": 0.89,
      "evidence": "frame_00005.jpg"
    },
    {
      "source": "audio",
      "category": "hate_speech",
      "start_sec": 12.4,
      "end_sec": 15.8,
      "peak_score": 0.93,
      "evidence": "đoạn nội dung toxic được phát hiện..."
    }
  ]
}
```

#### Response `200 OK` — an toàn

```json
{
  "video_id": "550e8400-e29b-41d4-a716-446655440000",
  "video_url": "http://localhost:9000/videos/550e8400-e29b-41d4-a716-446655440000.mp4?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
  "status": "completed",
  "stage": "completed",
  "progress_percent": 100,
  "original_filename": "clip.mp4",
  "uploaded_at": "2026-05-23T10:00:00Z",
  "processed_at": "2026-05-23T10:05:30Z",
  "verdict": {
    "verdict": "safe",
    "violated": false,
    "risk_score": 0.05,
    "final_score": 0.05,
    "frame_score": 0.04,
    "audio_score": 0.08,
    "total_frames": 100,
    "video_duration_sec": 95.0,
    "hard_rule_triggered": false
  },
  "violation_segments": []
}
```

#### Response `200 OK` — thất bại

```json
{
  "video_id": "550e8400-e29b-41d4-a716-446655440000",
  "video_url": "http://localhost:9000/videos/550e8400-e29b-41d4-a716-446655440000.mp4?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
  "status": "failed",
  "stage": "failed",
  "progress_percent": 0,
  "original_filename": "clip.mp4",
  "uploaded_at": "2026-05-23T10:00:00Z",
  "processed_at": "2026-05-23T10:01:00Z"
}
```

**Response fields — root:**

| Field | Type | Khi nào có | Mô tả |
|-------|------|------------|-------|
| `video_id` | string | Luôn | UUID video |
| `video_url` | string | Luôn có (có thể `""` nếu presign lỗi) | Presigned URL MinIO; hết hạn sau `MINIO_PRESIGN_TTL` — gọi lại API để refresh |
| `status` | string | Luôn | Trạng thái xử lý |
| `stage` | string | Luôn | Stage pipeline |
| `progress_percent` | int | Luôn | 0–100 |
| `original_filename` | string | Luôn | Tên file gốc |
| `uploaded_at` | ISO 8601 | Luôn | Thời điểm upload |
| `processed_at` | ISO 8601 \| null | `completed` / `failed` | Thời điểm kết thúc |
| `verdict` | object | `status == completed` | Tóm tắt kết quả moderation |
| `violation_segments` | array | `status == completed` | Có thể rỗng `[]` |

**Response fields — `verdict`:**

| Field | Type | Mô tả |
|-------|------|-------|
| `verdict` | string | `safe` \| `warning` \| `violation` |
| `violated` | boolean | `verdict != "safe"` (gồm cả `warning`) |
| `risk_score` | float | Alias của `final_score` (0.0–1.0) |
| `final_score` | float | `max(α×frame + β×audio, audio_score)` — không bị pha loãng bởi video dài |
| `frame_score` | float | Điểm visual (0–1), trung bình weighted trên tất cả frame |
| `audio_score` | float | `max(coverage×conf, density)` — coverage = tỷ lệ giây toxic / duration |
| `total_frames` | int | Số frame đã phân tích |
| `video_duration_sec` | float | Duration dùng normalize audio |
| `hard_rule_triggered` | boolean | Có kích hoạt rule cứng |
| `hard_rule_reason` | string | Mã rule (xem bảng dưới); rỗng nếu không trigger |
| `transcript` | string | Toàn bộ lời thoại Whisper (có thể rỗng) |

**Ngưỡng verdict theo `final_score` (mặc định):**

| `final_score` | `verdict` | `violated` |
|---------------|-----------|------------|
| `< 0.3` | `safe` | `false` |
| `0.3 – 0.6` | `warning` | `true` |
| `≥ 0.6` | `violation` | `true` |

**Hard rules (bypass ngưỡng → luôn `violation`):**

| `hard_rule_reason` | Điều kiện (mặc định) |
|--------------------|----------------------|
| `nsfw_sustained_5s` | Frame `nsfw`, conf ≥ 0.98, liên tục ≥ 5s |
| `violence_consecutive_frames` | ≥ 10 frame `violence` liên tiếp |
| `toxic_sustained_15s` | Audio toxic liên tục ≥ 15s (gap merge 0.5s) |
| `toxic_many_segments` | ≥ 8 câu `Toxic`, conf trung bình ≥ 0.85 |
| `toxic_total_duration` | Tổng giây toxic (merged) ≥ 45s |
| `toxic_coverage_ratio` | Toxic chiếm ≥ 15% thời lượng video |

> Video đã xử lý trước khi đổi logic cần **upload/xử lý lại** để cập nhật verdict trong DB.

**Response fields — `violation_segments[]`:**

| Field | Type | Mô tả |
|-------|------|-------|
| `source` | string | `visual` \| `audio` |
| `category` | string | `nudity`, `violence`, `hate_speech` |
| `start_sec` | float | Giây bắt đầu |
| `end_sec` | float | Giây kết thúc |
| `peak_score` | float | Score cao nhất trong đoạn |
| `evidence` | string | Frame filename hoặc đoạn text (max ~200 ký tự) |

**Lỗi thường gặp:**

| HTTP | `message` |
|------|-----------|
| `400` | `invalid video id` |
| `401` | Token errors |
| `404` | `video not found` (không tồn tại hoặc không thuộc user) |
| `500` | `failed to load video` |

---

## 6. WebSocket — `/api/v1/ws/pipeline`

Realtime cập nhật tiến độ xử lý video qua WebSocket.

### 6.1 Kết nối

```http
GET /api/v1/ws/pipeline?token=<access_token>
Upgrade: websocket
```

**Auth:** Access token qua một trong hai cách:
- Query: `?token=<access_token>` (khuyến nghị cho browser)
- Header: `Authorization: Bearer <access_token>`

**Upgrade thành công:** HTTP `101 Switching Protocols`

**Lỗi trước khi upgrade (JSON):**

| HTTP | `message` |
|------|-----------|
| `401` | `missing access token`, `invalid token`, `token revoked` |
| `500` | `failed to validate token`, `websocket upgrade failed` |

### 6.2 Message server → client

Server **chỉ push** (client không cần gửi message nghiệp vụ). Mỗi event là JSON text frame:

```json
{
  "type": "video.progress",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "video_id": "660e8400-e29b-41d4-a716-446655440001",
  "status": "processing",
  "stage": "frame_analysis",
  "progress_percent": 50
}
```

| Field | Type | Mô tả |
|-------|------|-------|
| `type` | string | Luôn `video.progress` |
| `user_id` | string | UUID owner video |
| `video_id` | string | UUID video |
| `status` | string | `processing` \| `completed` \| `failed` |
| `stage` | string | Stage pipeline |
| `progress_percent` | int | 0–100 |

**Khi nào nhận event:**
- Mỗi lần worker đổi stage (`starting` → … → `aggregation`)
- Khi video `completed` (`stage=completed`, `progress_percent=100`)
- Khi video `failed` (`stage=failed`, `progress_percent=0`)

**Lọc theo user:** Hub chỉ gửi event tới connection có `user_id` khớp `event.user_id`.

### 6.3 Ví dụ client (JavaScript)

```javascript
const token = localStorage.getItem("access_token");
const ws = new WebSocket(`ws://localhost:8080/api/v1/ws/pipeline?token=${token}`);

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  if (data.type === "video.progress") {
    console.log(data.video_id, data.stage, data.progress_percent);
  }
};

ws.onerror = (err) => console.error("WS error", err);
ws.onclose = () => console.log("WS closed");
```

### 6.4 Keep-alive

Server gửi WebSocket ping định kỳ; client cần trả pong (browser tự xử lý).

---

## 7. Tham chiếu enum & giá trị

### 7.1 Video `status`

| Giá trị | Mô tả |
|---------|-------|
| `uploaded` | Vừa tạo record, đang upload MinIO |
| `processing` | Đang xử lý pipeline |
| `completed` | Xử lý xong, có verdict |
| `failed` | Lỗi upload hoặc pipeline |

### 7.2 Video `stage`

| Stage | `progress_percent` |
|-------|-------------------|
| `starting` | 0 |
| `frame_extraction` | 15 |
| `audio_extraction` | 35 |
| `frame_analysis` | 50 |
| `audio_analysis` | 65 |
| `aggregation` | 90 |
| `completed` | 100 |
| `failed` | 0 |

### 7.3 Verdict (`verdict.verdict`)

| Giá trị | Ý nghĩa | `violated` |
|---------|---------|------------|
| `safe` | Không vi phạm (điểm thấp, không hard rule) | `false` |
| `warning` | Nghi ngờ / cần review (điểm trung bình) | `true` |
| `violation` | Vi phạm rõ (điểm cao hoặc hard rule) | `true` |

**Lưu ý:** `violation_segments` có thể không rỗng trong khi `verdict` vẫn `safe` nếu chỉ vài đoạn ngắn và chưa vượt ngưỡng — ưu tiên hiển thị cả `verdict` lẫn timeline.

### 7.4 Violation segment (`violation_segments[]`)

| `source` | `category` | Nguồn AI |
|----------|------------|----------|
| `visual` | `nudity` | Frame label `nsfw` |
| `visual` | `violence` | Frame label `violence` |
| `audio` | `hate_speech` | Câu label `Toxic` (Whisper + PhoBERT) |

### 7.5 Biến môi trường moderation & MinIO

| Biến | Mặc định | Mô tả |
|------|----------|--------|
| `MOD_FRAME_WEIGHT` | `0.7` | Trọng số visual trong fusion |
| `MOD_AUDIO_WEIGHT` | `0.3` | Trọng số audio trong fusion |
| `MOD_SAFE_THRESHOLD` | `0.3` | Dưới ngưỡng → `safe` |
| `MOD_VIOLATION_THRESHOLD` | `0.6` | Từ ngưỡng → `violation` |
| `MOD_MAX_LABEL_WEIGHT` | `5` | Chuẩn hóa điểm (violence = 5) |
| `MOD_HARD_NSFW_CONF` | `0.98` | Hard rule NSFW |
| `MOD_HARD_NSFW_SEC` | `5` | Giây NSFW liên tục |
| `MOD_HARD_VIOLENCE_FRAMES` | `10` | Frame violence liên tiếp |
| `MOD_HARD_TOXIC_SEC` | `15` | Giây toxic liên tục |
| `MOD_HARD_TOXIC_COVERAGE` | `0.15` | Tỷ lệ thời lượng toxic |
| `MOD_HARD_TOXIC_SEGMENTS` | `8` | Số câu toxic |
| `MOD_HARD_TOXIC_TOTAL_SEC` | `45` | Tổng giây toxic merged |
| `MINIO_ENDPOINT` | `minio:9000` | Upload/download trong Docker network |
| `MINIO_PUBLIC_ENDPOINT` | `http://localhost:9000` | Host trong presigned URL (browser). Trong Docker, API vẫn kết nối `MINIO_ENDPOINT` để ký |
| `MINIO_PRESIGN_TTL` | `1h` | TTL `video_url` |
| `MINIO_API_CORS_ALLOW_ORIGIN` | `*` (trên container `minio`) | CORS toàn cluster cho browser |

**Phát video trên FE:**

```html
<video controls src={video_url} />
```

Presign ký trên `MINIO_PUBLIC_ENDPOINT` (không đổi host sau khi ký). CORS: `MINIO_API_CORS_ALLOW_ORIGIN` trên service `minio` (Community không hỗ trợ `mc cors set` theo bucket).

**Lỗi thường gặp:** `AccessDenied` → sai `MINIO_PUBLIC_ENDPOINT` hoặc object không tồn tại; CORS → restart `minio` sau khi đổi env.

---

## Phụ lục — Bảng tóm tắt endpoint

| Method | Path | Auth | Mô tả |
|--------|------|------|-------|
| `POST` | `/api/v1/auth/register` | ❌ | Đăng ký |
| `POST` | `/api/v1/auth/login` | ❌ | Đăng nhập |
| `POST` | `/api/v1/auth/google` | ❌ | Google login |
| `POST` | `/api/v1/auth/refresh` | ❌ | Refresh token |
| `POST` | `/api/v1/auth/logout` | ✅ | Logout |
| `GET` | `/api/v1/users/me` | ✅ | Profile |
| `PATCH` | `/api/v1/users/me` | ✅ | Cập nhật profile |
| `PATCH` | `/api/v1/users/me/password` | ✅ | Đổi mật khẩu |
| `GET` | `/api/v1/videos` | ✅ | Danh sách video |
| `POST` | `/api/v1/videos/upload` | ✅ | Upload video |
| `GET` | `/api/v1/videos/:id/status` | ✅ | Trạng thái + kết quả |
| `GET` | `/api/v1/ws/pipeline` | token | WebSocket progress |

---

*Tài liệu đồng bộ với codebase tại `server/internal/app/server_route.go`. Cập nhật file này khi thêm/sửa API.*
