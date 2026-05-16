package pkg

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheProvider interface {
	Set(key string, value interface{}, ttl time.Duration) error
	Get(key string) (string, error)
	IsExist(key string) (bool, error)
}

type redisCache struct {
	client *redis.Client
}

func NewCacheProvider(address, password string, db int) (CacheProvider, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       db,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return &redisCache{
		client: rdb,
	}, nil
}

func (r *redisCache) Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return r.client.Get(ctx, key).Result()
}

func (r *redisCache) Set(key string, value interface{}, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *redisCache) IsExist(key string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
