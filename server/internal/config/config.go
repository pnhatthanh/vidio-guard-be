package config

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"
)

type MinioConfig struct {
	Endpoint         string
	PublicEndpoint   string // browser-facing host for presigned URLs (e.g. http://localhost:9000)
	AccessKey        string
	SecretKey        string
	Bucket           string
	UseSSL           bool
	PresignURLTTL    time.Duration
}

type RedisConfig struct {
	Addr            string
	Password        string
	DB              int
	ProgressChannel string
}

type AsynqConfig struct {
	Queue       string
	Concurrency int
	MaxRetry    int
	TaskTimeout time.Duration
}

type StatusConfig struct {
	KeyPrefix string
	TTL       time.Duration
}

type ServerConfig struct {
	Addr string
}

type PostgresConfig struct {
	DSN string 
}

type JWTConfig struct {
	AccessSecret string
	AccessTTL    time.Duration 
	RefreshTTL   time.Duration
}

type GoogleConfig struct {
	ClientID string
}

type AIServiceConfig struct {
	FrameModeratorUrl string
	NSFWThreshold     float64
	ViolenceThreshold float64
	ChunkSize         int
	EarlyExitCount    int
	AudioModeratorUrl string
}

// ModerationConfig controls risk scoring, fusion weights, and hard rules.
type ModerationConfig struct {
	FrameWeight            float64
	AudioWeight            float64
	SafeThreshold          float64
	ViolationThreshold     float64
	MaxLabelWeight         float64
	HardNsfwConfidence     float64
	HardNsfwSec            float64
	HardViolenceFrames     int
	HardToxicSec           float64
	HardToxicCoverageRatio float64 // total toxic time / video duration → violation
	HardToxicSegmentCount  int     // flagged sentences count → violation
	HardToxicTotalSec      float64 // merged toxic duration → violation
}

type Config struct {
	Server    ServerConfig
	OutputDir string

	Postgres PostgresConfig
	JWT      JWTConfig
	Google   GoogleConfig

	Minio     MinioConfig
	Redis     RedisConfig
	Asynq     AsynqConfig
	Status    StatusConfig
	AIService   AIServiceConfig
	Moderation  ModerationConfig
}

func Load() (*Config, error) {
	concurrency := runtime.NumCPU()
	if concurrency < 1 {
		concurrency = 1
	}

	cfg := &Config{
		Server: ServerConfig{
			Addr: getenv("HTTP_ADDR", ":8080"),
		},
		OutputDir: getenv("OUTPUT_DIR", "outputs"),

		Postgres: PostgresConfig{
			DSN: getenv("POSTGRES_DSN", "postgres://postgres:postgres@postgres:5432/vidioguard?sslmode=disable"),
		},
		JWT: JWTConfig{
			AccessSecret: getenv("JWT_ACCESS_SECRET", "your-access-secret-change-me"),
			AccessTTL:    getenvDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL:   getenvDuration("JWT_REFRESH_TTL", 168*time.Hour),
		},
		Google: GoogleConfig{
			ClientID: getenv("GOOGLE_CLIENT_ID", ""),
		},

		Minio: MinioConfig{
			Endpoint:       getenv("MINIO_ENDPOINT", "minio:9000"),
			PublicEndpoint: getenv("MINIO_PUBLIC_ENDPOINT", "http://localhost:9000"),
			AccessKey:      getenv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey:      getenv("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:         getenv("MINIO_BUCKET", "videos"),
			UseSSL:         getenvBool("MINIO_USE_SSL", false),
			PresignURLTTL:  getenvDuration("MINIO_PRESIGN_TTL", time.Hour),
		},
		Redis: RedisConfig{
			Addr:            getenv("REDIS_ADDR", "redis:6379"),
			Password:        getenv("REDIS_PASSWORD", ""),
			DB:              getenvInt("REDIS_DB", 0),
			ProgressChannel: getenv("REDIS_PROGRESS_CHANNEL", "videoguard:video:progress"),
		},
		Asynq: AsynqConfig{
			Queue:       getenv("ASYNQ_QUEUE", "video"),
			Concurrency: getenvInt("ASYNQ_CONCURRENCY", concurrency),
			MaxRetry:    getenvInt("ASYNQ_MAX_RETRY", 10),
			TaskTimeout: getenvDuration("ASYNQ_TASK_TIMEOUT", 30*time.Minute),
		},
		Status: StatusConfig{
			KeyPrefix: getenv("STATUS_KEY_PREFIX", "status:"),
			TTL:       getenvDuration("STATUS_TTL", 24*time.Hour),
		},
		AIService: AIServiceConfig{
			FrameModeratorUrl: getenv("AI_FRAME_MODERATOR_URL", "http://image-moderation:8000"),
			NSFWThreshold:     getenvFloat("AI_NSFW_THRESHOLD", 0.6),
			ViolenceThreshold: getenvFloat("AI_VIOLENCE_THRESHOLD", 0.6),
			ChunkSize:         getenvInt("AI_CHUNK_SIZE", 32),
			EarlyExitCount:    getenvInt("AI_EARLY_EXIT_COUNT", 3),
			AudioModeratorUrl: getenv("AI_AUDIO_MODERATOR_URL", "http://audio-moderation:8000"),
		},
		Moderation: ModerationConfig{
			FrameWeight:        getenvFloat("MOD_FRAME_WEIGHT", 0.7),
			AudioWeight:        getenvFloat("MOD_AUDIO_WEIGHT", 0.3),
			SafeThreshold:      getenvFloat("MOD_SAFE_THRESHOLD", 0.3),
			ViolationThreshold: getenvFloat("MOD_VIOLATION_THRESHOLD", 0.6),
			MaxLabelWeight:     getenvFloat("MOD_MAX_LABEL_WEIGHT", 5),
			HardNsfwConfidence: getenvFloat("MOD_HARD_NSFW_CONF", 0.98),
			HardNsfwSec:        getenvFloat("MOD_HARD_NSFW_SEC", 5),
			HardViolenceFrames:     getenvInt("MOD_HARD_VIOLENCE_FRAMES", 10),
			HardToxicSec:           getenvFloat("MOD_HARD_TOXIC_SEC", 15),
			HardToxicCoverageRatio: getenvFloat("MOD_HARD_TOXIC_COVERAGE", 0.15),
			HardToxicSegmentCount:  getenvInt("MOD_HARD_TOXIC_SEGMENTS", 8),
			HardToxicTotalSec:      getenvFloat("MOD_HARD_TOXIC_TOTAL_SEC", 45),
		},
	}

	if cfg.Asynq.Concurrency < 1 {
		return &Config{}, fmt.Errorf("ASYNQ_CONCURRENCY must be >= 1")
	}
	if cfg.Asynq.MaxRetry < 0 {
		return &Config{}, fmt.Errorf("ASYNQ_MAX_RETRY must be >= 0")
	}
	if cfg.Asynq.Queue == "" {
		return &Config{}, fmt.Errorf("ASYNQ_QUEUE is required")
	}
	if cfg.Status.KeyPrefix == "" {
		return &Config{}, fmt.Errorf("STATUS_KEY_PREFIX is required")
	}
	if cfg.Status.TTL <= 0 {
		return &Config{}, fmt.Errorf("STATUS_TTL must be > 0")
	}
	if cfg.JWT.AccessTTL <= 0 {
		return &Config{}, fmt.Errorf("JWT_ACCESS_TTL must be > 0")
	}
	if cfg.JWT.RefreshTTL <= 0 {
		return &Config{}, fmt.Errorf("JWT_REFRESH_TTL must be > 0")
	}
	if cfg.Moderation.MaxLabelWeight <= 0 {
		return &Config{}, fmt.Errorf("MOD_MAX_LABEL_WEIGHT must be > 0")
	}
	if cfg.Moderation.SafeThreshold < 0 || cfg.Moderation.ViolationThreshold <= cfg.Moderation.SafeThreshold {
		return &Config{}, fmt.Errorf("MOD_SAFE_THRESHOLD must be < MOD_VIOLATION_THRESHOLD")
	}

	return cfg, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func getenvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

func getenvDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func getenvFloat(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}
