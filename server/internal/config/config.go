package config

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"
)

type MinioConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
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
	AIService AIServiceConfig
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
			Endpoint:  getenv("MINIO_ENDPOINT", "minio:9000"),
			AccessKey: getenv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey: getenv("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:    getenv("MINIO_BUCKET", "videos"),
			UseSSL:    getenvBool("MINIO_USE_SSL", false),
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
