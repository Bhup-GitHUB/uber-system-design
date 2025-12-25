package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"uber-system/pkg/models"
	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	ctx    context.Context
	ttl    time.Duration
}

func NewRedisCache(addr, password string, db int, ttl time.Duration) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client: client,
		ctx:    ctx,
		ttl:    ttl,
	}, nil
}

func (rc *RedisCache) AddDriver(driver *models.Driver, city string) error {
	key := fmt.Sprintf("drivers:%s", city)
	err := rc.client.GeoAdd(rc.ctx, key, &redis.GeoLocation{
		Name:      driver.ID,
		Longitude: driver.Location.Lng,
		Latitude:  driver.Location.Lat,
	}).Err()
	if err != nil {
		return err
	}

	driverData, err := json.Marshal(driver)
	if err != nil {
		return err
	}

	metaKey := fmt.Sprintf("driver:%s:meta", driver.ID)
	return rc.client.Set(rc.ctx, metaKey, driverData, rc.ttl).Err()
}

func (rc *RedisCache) UpdateLocation(driverID, city string, lat, lng float64) error {
	key := fmt.Sprintf("drivers:%s", city)
	return rc.client.GeoAdd(rc.ctx, key, &redis.GeoLocation{
		Name:      driverID,
		Longitude: lng,
		Latitude:  lat,
	}).Err()
}

func (rc *RedisCache) RemoveDriver(driverID, city string) error {
	key := fmt.Sprintf("drivers:%s", city)
	metaKey := fmt.Sprintf("driver:%s:meta", driverID)
	err := rc.client.ZRem(rc.ctx, key, driverID).Err()
	if err != nil {
		return err
	}
	return rc.client.Del(rc.ctx, metaKey).Err()
}

func (rc *RedisCache) SearchRadius(city string, lat, lng, radiusKm float64) ([]string, error) {
	key := fmt.Sprintf("drivers:%s", city)
	results, err := rc.client.GeoRadius(rc.ctx, key, lng, lat, &redis.GeoRadiusQuery{
		Radius:    radiusKm,
		Unit:      "km",
		WithCoord: true,
		WithDist:  true,
		Sort:      "ASC",
	}).Result()
	if err != nil {
		return nil, err
	}

	driverIDs := make([]string, 0, len(results))
	for _, result := range results {
		driverIDs = append(driverIDs, result.Name)
	}
	return driverIDs, nil
}

func (rc *RedisCache) GetDriver(driverID string) (*models.Driver, error) {
	metaKey := fmt.Sprintf("driver:%s:meta", driverID)
	data, err := rc.client.Get(rc.ctx, metaKey).Result()
	if err != nil {
		return nil, err
	}

	var driver models.Driver
	if err := json.Unmarshal([]byte(data), &driver); err != nil {
		return nil, err
	}
	return &driver, nil
}

func (rc *RedisCache) GetStats(city string) (map[string]interface{}, error) {
	key := fmt.Sprintf("drivers:%s", city)
	count, err := rc.client.ZCard(rc.ctx, key).Result()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"city":          city,
		"driver_count":  count,
		"cache_enabled": true,
	}, nil
}

func (rc *RedisCache) Close() error {
	return rc.client.Close()
}

