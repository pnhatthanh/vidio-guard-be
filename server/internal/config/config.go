package config

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type MinioConfig struct {
	Endpoint       string
	PublicEndpoint string
	AccessKey      string
	SecretKey      string
	Bucket         string
	UseSSL         bool
	PresignURLTTL  time.Duration
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type AsynqConfig struct {
	Queue       string
	Concurrency int
	MaxRetry    int
	TaskTimeout time.Duration
}

type ServerConfig struct {
	Addr      string
	OutputDir string
}

type PostgresConfig struct {
	DSN             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

type JWTConfig struct {
	AccessSecret string
	AccessTTL    time.Duration
	RefreshTTL   time.Duration
}

type GoogleConfig struct {
	ClientID string
}

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	UseTLS   bool
}

type PasswordResetConfig struct {
	ResetPageURL string
	OTPTTL       time.Duration
	CooldownTTL  time.Duration
	MaxAttempts  int
	AttemptsTTL  time.Duration
}

type AIServiceConfig struct {
	FrameModeratorUrl string
	NSFWThreshold     float64
	ViolenceThreshold float64
	ChunkSize         int
	EarlyExitCount    int
	AudioModeratorUrl string
	AudioTaskTimeout  time.Duration
}

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
	Server ServerConfig

	Postgres PostgresConfig
	JWT      JWTConfig
	Google   GoogleConfig

	Minio         MinioConfig
	Redis         RedisConfig
	Asynq         AsynqConfig
	AIService     AIServiceConfig
	Moderation    ModerationConfig
	SMTP          SMTPConfig
	PasswordReset PasswordResetConfig
}

func Load() (*Config, error) {
	concurrency := runtime.NumCPU()
	if concurrency < 1 {
		concurrency = 1
	}

	cfg := &Config{
		Server: ServerConfig{
			Addr:      getenv("HTTP_ADDR", ":8080"),
			OutputDir: getenv("OUTPUT_DIR", "outputs"),
		},

		Postgres: PostgresConfig{
			DSN:             getenv("POSTGRES_DSN", "postgres://postgres:postgres@postgres:5432/vidioguard?sslmode=disable"),
			MaxIdleConns:    getenvInt("POSTGRES_MAX_IDLE_CONNS", 10),
			MaxOpenConns:    getenvInt("POSTGRES_MAX_OPEN_CONNS", 100),
			ConnMaxLifetime: getenvDuration("POSTGRES_CONN_MAX_LIFETIME", time.Hour),
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
			Addr:     getenv("REDIS_ADDR", "redis:6379"),
			Password: getenv("REDIS_PASSWORD", ""),
			DB:       getenvInt("REDIS_DB", 0),
		},
		Asynq: AsynqConfig{
			Queue:       getenv("ASYNQ_QUEUE", "video"),
			Concurrency: getenvInt("ASYNQ_CONCURRENCY", concurrency),
			MaxRetry:    getenvInt("ASYNQ_MAX_RETRY", 10),
			TaskTimeout: getenvDuration("ASYNQ_TASK_TIMEOUT", 30*time.Minute),
		},
		AIService: AIServiceConfig{
			FrameModeratorUrl: getenv("AI_FRAME_MODERATOR_URL", "http://image-moderation:8000"),
			NSFWThreshold:     getenvFloat("AI_NSFW_THRESHOLD", 0.6),
			ViolenceThreshold: getenvFloat("AI_VIOLENCE_THRESHOLD", 0.6),
			ChunkSize:         getenvInt("AI_CHUNK_SIZE", 32),
			EarlyExitCount:    getenvInt("AI_EARLY_EXIT_COUNT", 3),
			AudioModeratorUrl: getenv("AI_AUDIO_MODERATOR_URL", "http://audio-moderation:8001"),
			AudioTaskTimeout:  getenvDuration("AI_AUDIO_TASK_TIMEOUT", 20*time.Minute),
		},
		Moderation: ModerationConfig{
			FrameWeight:            getenvFloat("MOD_FRAME_WEIGHT", 0.7),
			AudioWeight:            getenvFloat("MOD_AUDIO_WEIGHT", 0.3),
			SafeThreshold:          getenvFloat("MOD_SAFE_THRESHOLD", 0.3),
			ViolationThreshold:     getenvFloat("MOD_VIOLATION_THRESHOLD", 0.6),
			MaxLabelWeight:         getenvFloat("MOD_MAX_LABEL_WEIGHT", 5),
			HardNsfwConfidence:     getenvFloat("MOD_HARD_NSFW_CONF", 0.90),
			HardNsfwSec:            getenvFloat("MOD_HARD_NSFW_SEC", 5),
			HardViolenceFrames:     getenvInt("MOD_HARD_VIOLENCE_FRAMES", 10),
			HardToxicSec:           getenvFloat("MOD_HARD_TOXIC_SEC", 15),
			HardToxicCoverageRatio: getenvFloat("MOD_HARD_TOXIC_COVERAGE", 0.15),
			HardToxicSegmentCount:  getenvInt("MOD_HARD_TOXIC_SEGMENTS", 8),
			HardToxicTotalSec:      getenvFloat("MOD_HARD_TOXIC_TOTAL_SEC", 30),
		},
		SMTP: SMTPConfig{
			Host:     strings.TrimSpace(getenv("SMTP_HOST", "")),
			Port:     getenvInt("SMTP_PORT", 587),
			User:     getenv("SMTP_USER", ""),
			Password: getenv("SMTP_PASSWORD", ""),
			From:     strings.TrimSpace(getenv("SMTP_FROM", "")),
			UseTLS:   getenvBool("SMTP_USE_TLS", true),
		},
		PasswordReset: PasswordResetConfig{
			ResetPageURL: strings.TrimRight(strings.TrimSpace(getenv("PWD_RESET_PAGE_URL", "http://localhost:3000/reset-password")), "/"),
			OTPTTL:       getenvDuration("PWD_RESET_OTP_TTL", 15*time.Minute),
			CooldownTTL:  getenvDuration("PWD_RESET_COOLDOWN", 60*time.Second),
			MaxAttempts:  getenvInt("PWD_RESET_MAX_ATTEMPTS", 5),
			AttemptsTTL:  getenvDuration("PWD_RESET_ATTEMPTS_TTL", 30*time.Minute),
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
