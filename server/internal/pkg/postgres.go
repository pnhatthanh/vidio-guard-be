package pkg

import (
	"context"
	"fmt"
	"time"

	"github.com/pnhatthanh/vidio-guard-be/internal/config"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DBProvider interface {
	DB() *gorm.DB
	Close() error
}

type postgresDB struct {
	db *gorm.DB
}

func NewDBProvider(cfg *config.PostgresConfig) (DBProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("postgres config is required")
	}
	if cfg.DSN == "" {
		return nil, fmt.Errorf("POSTGRES_DSN is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	db.AutoMigrate(&model.User{}, &model.RefreshToken{}, &model.Video{},
		&model.FinalVerdict{}, &model.ViolationSegment{})

	return &postgresDB{db: db}, nil
}

func (p *postgresDB) DB() *gorm.DB {
	return p.db
}

func (p *postgresDB) Close() error {
	if p == nil || p.db == nil {
		return nil
	}
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
