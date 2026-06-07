package cache

import (
	"context"
	"diabetify/internal/config"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisClient() (*RedisClient, error) {
	opt, err := redisOptionsFromEnv()
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)
	ctx := context.Background()

	// Test connection
	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{
		client: client,
		ctx:    ctx,
	}, nil
}

func redisOptionsFromEnv() (*redis.Options, error) {
	settings := config.Load()
	if settings.Redis.URL != "" {
		opt, err := redis.ParseURL(settings.Redis.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse REDIS_URL: %w", err)
		}
		return opt, nil
	}

	return &redis.Options{
		Addr:     fmt.Sprintf("%s:%s", settings.Redis.Host, settings.Redis.Port),
		Password: settings.Redis.Password,
		DB:       settings.Redis.DB,
	}, nil
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

const activeChallengerKey = "shadow:active_challenger"

type ChallengerInfo struct {
	DeploymentID int    `json:"deployment_id"`
	ModelID      int    `json:"model_id"`
	ModelVersion string `json:"model_version"`
}

// GetActiveChallengerInfo returns nil, nil when no active challenger exists.
func (r *RedisClient) GetActiveChallengerInfo() (*ChallengerInfo, error) {
	data, err := r.client.Get(r.ctx, activeChallengerKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get challenger info: %w", err)
	}
	var info ChallengerInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal challenger info: %w", err)
	}
	return &info, nil
}

// Get Redis status
func (r *RedisClient) GetStatus() (map[string]interface{}, error) {
	info, err := r.client.Info(r.ctx).Result()
	if err != nil {
		return nil, err
	}

	stats := r.client.PoolStats()

	return map[string]interface{}{
		"connected":    true,
		"hits":         stats.Hits,
		"misses":       stats.Misses,
		"active_conns": stats.TotalConns,
		"redis_info":   info,
	}, nil
}
