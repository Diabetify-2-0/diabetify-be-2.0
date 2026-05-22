package cache

import (
	"context"
	"diabetify/internal/config"
	"encoding/json"
	"fmt"
	"time"

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

// Store what-if result with expiration
func (r *RedisClient) StoreWhatIfResult(jobID string, result map[string]interface{}, duration time.Duration) error {
	key := fmt.Sprintf("whatif:%s", jobID)

	// Add metadata
	result["stored_at"] = time.Now().Unix()
	result["expires_at"] = time.Now().Add(duration).Unix()

	jsonData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	err = r.client.Set(r.ctx, key, jsonData, duration).Err()
	if err != nil {
		return fmt.Errorf("failed to store result in Redis: %w", err)
	}

	return nil
}

// Get what-if result
func (r *RedisClient) GetWhatIfResult(jobID string) (map[string]interface{}, bool, error) {
	key := fmt.Sprintf("whatif:%s", jobID)

	data, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil // Key doesn't exist
		}
		return nil, false, fmt.Errorf("failed to get result from Redis: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return result, true, nil
}

// Delete what-if result
func (r *RedisClient) DeleteWhatIfResult(jobID string) error {
	key := fmt.Sprintf("whatif:%s", jobID)
	return r.client.Del(r.ctx, key).Err()
}

const activeChallengerKey = "shadow:active_challenger"
const activeChallengerTTL = 60 * time.Second

type ChallengerInfo struct {
	DeploymentID int    `json:"deployment_id"`
	ModelID      int    `json:"model_id"`
	ModelVersion string `json:"model_version"`
}

func (r *RedisClient) SetActiveChallengerInfo(info *ChallengerInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal challenger info: %w", err)
	}
	return r.client.Set(r.ctx, activeChallengerKey, data, activeChallengerTTL).Err()
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

func (r *RedisClient) DeleteActiveChallengerInfo() error {
	return r.client.Del(r.ctx, activeChallengerKey).Err()
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
