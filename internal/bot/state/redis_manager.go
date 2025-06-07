package state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisManager manages user states using Redis
type RedisManager struct {
	client *redis.Client
}

// NewRedisManager creates a new Redis-based state manager
func NewRedisManager(redisHost, redisPort string) (*RedisManager, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password:     "", // no password
		DB:           0,  // default DB
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisManager{
		client: client,
	}, nil
}

// SetUserState sets the state for a user with TTL
func (m *RedisManager) SetUserState(userID int64, state string) {
	ctx := context.Background()
	key := fmt.Sprintf("user:%d:state", userID)
	// TTL 24 часа для автоочистки неактивных состояний
	m.client.Set(ctx, key, state, 24*time.Hour)
}

// GetUserState gets the state for a user
func (m *RedisManager) GetUserState(userID int64) string {
	ctx := context.Background()
	key := fmt.Sprintf("user:%d:state", userID)
	result := m.client.Get(ctx, key)
	if result.Err() == redis.Nil {
		return None // default state
	}
	if result.Err() != nil {
		return None // fallback on error
	}
	return result.Val()
}

// SetTempData sets temporary data for a user
func (m *RedisManager) SetTempData(userID int64, key string, value interface{}) {
	// Get current temp data
	tempData := m.getTempDataMap(userID)
	if tempData == nil {
		tempData = make(map[string]interface{})
	}

	// Update the specific key
	tempData[key] = value

	// Save back to Redis
	m.saveTempDataMap(userID, tempData)
}

// GetTempData gets temporary data for a user
func (m *RedisManager) GetTempData(userID int64, key string) (interface{}, bool) {
	tempData := m.getTempDataMap(userID)
	if tempData == nil {
		return nil, false
	}

	value, exists := tempData[key]
	return value, exists
}

// ClearTempData clears all temporary data for a user
func (m *RedisManager) ClearTempData(userID int64) {
	ctx := context.Background()
	key := fmt.Sprintf("user:%d:temp", userID)
	m.client.Del(ctx, key)
}

// SetUserWeight sets the weight for a user (if needed)
func (m *RedisManager) SetUserWeight(userID int64, weight float64) {
	ctx := context.Background()
	key := fmt.Sprintf("user:%d:weight", userID)
	m.client.Set(ctx, key, weight, 24*time.Hour)
}

// GetUserWeight gets the weight for a user (if needed)
func (m *RedisManager) GetUserWeight(userID int64) float64 {
	ctx := context.Background()
	key := fmt.Sprintf("user:%d:weight", userID)
	result := m.client.Get(ctx, key)
	if result.Err() != nil {
		return 0
	}
	weight, _ := result.Float64()
	return weight
}

// Close closes the Redis connection
func (m *RedisManager) Close() error {
	return m.client.Close()
}

// Helper methods
func (m *RedisManager) getTempDataMap(userID int64) map[string]interface{} {
	ctx := context.Background()
	key := fmt.Sprintf("user:%d:temp", userID)

	result := m.client.Get(ctx, key)
	if result.Err() == redis.Nil {
		return nil
	}
	if result.Err() != nil {
		return nil
	}

	var tempData map[string]interface{}
	if err := json.Unmarshal([]byte(result.Val()), &tempData); err != nil {
		return nil
	}

	return tempData
}

func (m *RedisManager) saveTempDataMap(userID int64, tempData map[string]interface{}) {
	ctx := context.Background()
	key := fmt.Sprintf("user:%d:temp", userID)

	data, err := json.Marshal(tempData)
	if err != nil {
		return
	}

	// TTL 24 часа
	m.client.Set(ctx, key, data, 24*time.Hour)
}
