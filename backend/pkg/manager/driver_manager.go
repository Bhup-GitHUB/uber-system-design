package manager

import (
	"fmt"
	"sync"
	"time"
	"uber-system/pkg/cache"
	"uber-system/pkg/geospatial"
	"uber-system/pkg/models"
	"uber-system/pkg/router"
)

type IndexType string

const (
	IndexTypeQuadTree IndexType = "quadtree"
	IndexTypeGrid     IndexType = "grid"
	IndexTypeRedis    IndexType = "redis"
)

type DriverManager struct {
	quadTree   *geospatial.QuadTree
	gridIndex  *geospatial.GridIndex
	redisCache *cache.RedisCache
	geoRouter  *router.GeoRouter
	drivers    map[string]*models.Driver
	mu         sync.RWMutex
	useRedis   bool
}

func NewDriverManager(minLat, maxLat, minLng, maxLng float64, redisAddr string, useRedis bool) (*DriverManager, error) {
	manager := &DriverManager{
		quadTree:  geospatial.NewQuadTree(minLat, maxLat, minLng, maxLng),
		gridIndex: geospatial.NewGridIndex(minLat, maxLat, minLng, maxLng, 0.5),
		geoRouter: router.NewGeoRouter(),
		drivers:   make(map[string]*models.Driver),
		useRedis:  useRedis,
	}

	if useRedis {
		redisCache, err := cache.NewRedisCache(redisAddr, "", 0, 30*time.Minute)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Redis: %w", err)
		}
		manager.redisCache = redisCache
	}

	manager.geoRouter.RegisterCity("mumbai", minLat, maxLat, minLng, maxLng)
	manager.geoRouter.RegisterCity("delhi", 28.3949, 28.8836, 76.8389, 77.3456)
	manager.geoRouter.RegisterCity("bangalore", 12.8342, 13.1476, 77.4577, 77.7878)
	manager.geoRouter.RegisterCity("hyderabad", 17.2403, 17.6868, 78.1636, 78.6569)
	manager.geoRouter.RegisterCity("chennai", 12.7948, 13.2402, 80.0889, 80.3044)
	return manager, nil
}

func (dm *DriverManager) AddDriver(driver *models.Driver) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	driver.UpdatedAt = time.Now()
	dm.drivers[driver.ID] = driver

	if !dm.quadTree.Insert(driver) {
		return fmt.Errorf("failed to insert into QuadTree")
	}

	if err := dm.gridIndex.Insert(driver); err != nil {
		return fmt.Errorf("failed to insert into Grid: %w", err)
	}

	if dm.useRedis && dm.redisCache != nil {
		city, _ := dm.geoRouter.GetCity(driver.Location.Lat, driver.Location.Lng)
		if city == "" {
			city = "mumbai"
		}
		if err := dm.redisCache.AddDriver(driver, city); err != nil {
			fmt.Printf("Redis cache error (non-fatal): %v\n", err)
		}
	}

	return nil
}

func (dm *DriverManager) UpdateLocation(driverID string, lat, lng float64) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	driver, exists := dm.drivers[driverID]
	if !exists {
		return fmt.Errorf("driver not found: %s", driverID)
	}

	oldLat, oldLng := driver.Location.Lat, driver.Location.Lng
	dm.quadTree.Remove(driverID)
	dm.gridIndex.Remove(driverID, oldLat, oldLng)

	driver.Location.Lat = lat
	driver.Location.Lng = lng
	driver.UpdatedAt = time.Now()

	dm.quadTree.Insert(driver)
	dm.gridIndex.Insert(driver)

	if dm.useRedis && dm.redisCache != nil {
		city, _ := dm.geoRouter.GetCity(lat, lng)
		if city == "" {
			city = "mumbai"
		}
		dm.redisCache.UpdateLocation(driverID, city, lat, lng)
	}

	return nil
}

func (dm *DriverManager) UpdateStatus(driverID, status string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	driver, exists := dm.drivers[driverID]
	if !exists {
		return fmt.Errorf("driver not found: %s", driverID)
	}

	driver.Status = status
	driver.UpdatedAt = time.Now()
	return nil
}

func (dm *DriverManager) SearchWithIndex(lat, lng, radiusKm float64, indexType IndexType) ([]models.DriverWithDistance, time.Duration, error) {
	startTime := time.Now()
	var drivers []*models.Driver

	switch indexType {
	case IndexTypeQuadTree:
		drivers = dm.quadTree.SearchRadius(lat, lng, radiusKm)
	case IndexTypeGrid:
		drivers = dm.gridIndex.SearchRadius(lat, lng, radiusKm)
	case IndexTypeRedis:
		if !dm.useRedis || dm.redisCache == nil {
			return nil, 0, fmt.Errorf("Redis not enabled")
		}
		city, _ := dm.geoRouter.GetCity(lat, lng)
		if city == "" {
			city = "mumbai"
		}
		driverIDs, err := dm.redisCache.SearchRadius(city, lat, lng, radiusKm)
		if err != nil {
			return nil, 0, err
		}
		drivers = make([]*models.Driver, 0, len(driverIDs))
		for _, id := range driverIDs {
			if driver, exists := dm.drivers[id]; exists {
				drivers = append(drivers, driver)
			}
		}
	default:
		return nil, 0, fmt.Errorf("unknown index type: %s", indexType)
	}

	results := make([]models.DriverWithDistance, 0)
	for _, driver := range drivers {
		if driver.Status != "available" {
			continue
		}

		distance := geospatial.Haversine(lat, lng, driver.Location.Lat, driver.Location.Lng)
		if distance <= radiusKm {
			results = append(results, models.DriverWithDistance{
				Driver:   *driver,
				Distance: distance,
			})
		}
	}

	for i := 0; i < len(results)-1; i++ {
		for j := 0; j < len(results)-i-1; j++ {
			if results[j].Distance > results[j+1].Distance {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}

	duration := time.Since(startTime)
	return results, duration, nil
}

func (dm *DriverManager) CompareIndexes(lat, lng, radiusKm float64) models.ComparisonResult {
	comparison := models.ComparisonResult{}

	qtResults, qtDuration, _ := dm.SearchWithIndex(lat, lng, radiusKm, IndexTypeQuadTree)
	comparison.QuadTree = map[string]interface{}{
		"count":    len(qtResults),
		"duration": qtDuration.String(),
	}

	gridResults, gridDuration, _ := dm.SearchWithIndex(lat, lng, radiusKm, IndexTypeGrid)
	comparison.Grid = map[string]interface{}{
		"count":    len(gridResults),
		"duration": gridDuration.String(),
	}

	if dm.useRedis && dm.redisCache != nil {
		redisResults, redisDuration, err := dm.SearchWithIndex(lat, lng, radiusKm, IndexTypeRedis)
		if err == nil {
			comparison.Redis = map[string]interface{}{
				"count":    len(redisResults),
				"duration": redisDuration.String(),
			}
		}
	}

	return comparison
}

func (dm *DriverManager) GetStats() map[string]interface{} {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	available := 0
	busy := 0
	offline := 0

	for _, driver := range dm.drivers {
		switch driver.Status {
		case "available":
			available++
		case "busy":
			busy++
		case "offline":
			offline++
		}
	}

	stats := map[string]interface{}{
		"total_drivers":     len(dm.drivers),
		"available_drivers": available,
		"busy_drivers":      busy,
		"offline_drivers":   offline,
		"grid_stats":        dm.gridIndex.GetStats(),
	}

	if dm.useRedis && dm.redisCache != nil {
		redisStats, _ := dm.redisCache.GetStats("mumbai")
		stats["redis_stats"] = redisStats
	}

	return stats
}

func (dm *DriverManager) Close() error {
	if dm.redisCache != nil {
		return dm.redisCache.Close()
	}
	return nil
}

