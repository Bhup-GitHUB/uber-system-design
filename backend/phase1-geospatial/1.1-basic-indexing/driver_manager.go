package basic

import (
	"fmt"
	"sync"
	"time"
	"uber-system/common"
)

type DriverManager struct {
	quadTree *QuadTree
	drivers  map[string]*common.Driver
	mu       sync.RWMutex
}

func NewDriverManager(minLat, maxLat, minLng, maxLng float64) *DriverManager {
	return &DriverManager{
		quadTree: NewQuadTree(minLat, maxLat, minLng, maxLng),
		drivers:  make(map[string]*common.Driver),
	}
}

func (dm *DriverManager) AddDriver(driver *common.Driver) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	driver.UpdatedAt = time.Now()

	dm.drivers[driver.ID] = driver

	if !dm.quadTree.Insert(driver) {
		delete(dm.drivers, driver.ID)
		return fmt.Errorf("failed to insert driver into quadtree")
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

	dm.quadTree.Remove(driverID)

	driver.Location.Lat = lat
	driver.Location.Lng = lng
	driver.UpdatedAt = time.Now()

	if !dm.quadTree.Insert(driver) {
		return fmt.Errorf("failed to re-insert driver into quadtree")
	}

	return nil
}

func (dm *DriverManager) SearchNearby(lat, lng, radiusKm float64) []common.DriverWithDistance {
	startTime := time.Now()
	drivers := dm.quadTree.SearchRadius(lat, lng, radiusKm)

	results := make([]common.DriverWithDistance, 0, len(drivers))
	for _, driver := range drivers {
		if driver.Status != "available" {
			continue
		}

		distance := Haversine(lat, lng, driver.Location.Lat, driver.Location.Lng)
		results = append(results, common.DriverWithDistance{
			Driver:   *driver,
			Distance: distance,
		})
	}

	for i := 0; i < len(results)-1; i++ {
		for j := 0; j < len(results)-i-1; j++ {
			if results[j].Distance > results[j+1].Distance {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}

	fmt.Printf("Search completed in %v\n", time.Since(startTime))

	return results
}

func (dm *DriverManager) GetDriver(driverID string) (*common.Driver, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	driver, exists := dm.drivers[driverID]
	if !exists {
		return nil, fmt.Errorf("driver not found: %s", driverID)
	}

	return driver, nil
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

	return map[string]interface{}{
		"total_drivers":     len(dm.drivers),
		"available_drivers": available,
		"busy_drivers":      busy,
		"offline_drivers":   offline,
	}
}

