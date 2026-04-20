package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
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
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse REDIS_URL: %w", err)
		}
		return opt, nil
	}

	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}

	db := 0
	if rawDB := os.Getenv("REDIS_DB"); rawDB != "" {
		parsedDB, err := strconv.Atoi(rawDB)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_DB %q: %w", rawDB, err)
		}
		db = parsedDB
	}

	return &redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
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
