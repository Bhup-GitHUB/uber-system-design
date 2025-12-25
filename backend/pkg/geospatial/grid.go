package geospatial

import (
	"fmt"
	"math"
	"sync"
	"uber-system/pkg/models"
)

type GridCell struct {
	Drivers map[string]*models.Driver
	mu      sync.RWMutex
}

type GridIndex struct {
	cells      map[string]*GridCell
	cellSizeKm float64
	boundary   BoundingBox
	mu         sync.RWMutex
}

func NewGridIndex(minLat, maxLat, minLng, maxLng, cellSizeKm float64) *GridIndex {
	return &GridIndex{
		cells:      make(map[string]*GridCell),
		cellSizeKm: cellSizeKm,
		boundary: BoundingBox{
			MinLat: minLat,
			MaxLat: maxLat,
			MinLng: minLng,
			MaxLng: maxLng,
		},
	}
}

func (gi *GridIndex) getCellKey(lat, lng float64) string {
	latDeg := gi.cellSizeKm / 111.0
	lngDeg := gi.cellSizeKm / (111.0 * math.Cos(lat*math.Pi/180))
	cellRow := int(math.Floor((lat - gi.boundary.MinLat) / latDeg))
	cellCol := int(math.Floor((lng - gi.boundary.MinLng) / lngDeg))
	return fmt.Sprintf("%d:%d", cellRow, cellCol)
}

func (gi *GridIndex) Insert(driver *models.Driver) error {
	key := gi.getCellKey(driver.Location.Lat, driver.Location.Lng)
	gi.mu.Lock()
	cell, exists := gi.cells[key]
	if !exists {
		cell = &GridCell{
			Drivers: make(map[string]*models.Driver),
		}
		gi.cells[key] = cell
	}
	gi.mu.Unlock()

	cell.mu.Lock()
	cell.Drivers[driver.ID] = driver
	cell.mu.Unlock()
	return nil
}

func (gi *GridIndex) Remove(driverID string, lat, lng float64) error {
	key := gi.getCellKey(lat, lng)
	gi.mu.RLock()
	cell, exists := gi.cells[key]
	gi.mu.RUnlock()

	if !exists {
		return fmt.Errorf("cell not found")
	}

	cell.mu.Lock()
	delete(cell.Drivers, driverID)
	cell.mu.Unlock()
	return nil
}

func (gi *GridIndex) SearchRadius(lat, lng, radiusKm float64) []*models.Driver {
	cellsToCheck := int(math.Ceil(radiusKm / gi.cellSizeKm))
	centerKey := gi.getCellKey(lat, lng)

	var centerRow, centerCol int
	fmt.Sscanf(centerKey, "%d:%d", &centerRow, &centerCol)

	results := make([]*models.Driver, 0)
	seen := make(map[string]bool)

	for row := centerRow - cellsToCheck; row <= centerRow+cellsToCheck; row++ {
		for col := centerCol - cellsToCheck; col <= centerCol+cellsToCheck; col++ {
			cellKey := fmt.Sprintf("%d:%d", row, col)
			gi.mu.RLock()
			cell, exists := gi.cells[cellKey]
			gi.mu.RUnlock()

			if !exists {
				continue
			}

			cell.mu.RLock()
			for _, driver := range cell.Drivers {
				if !seen[driver.ID] {
					results = append(results, driver)
					seen[driver.ID] = true
				}
			}
			cell.mu.RUnlock()
		}
	}

	return results
}

func (gi *GridIndex) GetStats() map[string]interface{} {
	gi.mu.RLock()
	defer gi.mu.RUnlock()

	totalDrivers := 0
	for _, cell := range gi.cells {
		cell.mu.RLock()
		totalDrivers += len(cell.Drivers)
		cell.mu.RUnlock()
	}

	return map[string]interface{}{
		"total_cells":   len(gi.cells),
		"total_drivers": totalDrivers,
		"cell_size_km":  gi.cellSizeKm,
	}
}

