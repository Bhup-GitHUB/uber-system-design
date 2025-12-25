package geospatial

import (
	"math"
	"sync"
	"uber-system/pkg/models"
)

const (
	MaxCapacity = 50
	MaxDepth    = 8
)

type QuadTreeNode struct {
	Boundary  BoundingBox
	Drivers   []*models.Driver
	Depth     int
	NorthWest *QuadTreeNode
	NorthEast *QuadTreeNode
	SouthWest *QuadTreeNode
	SouthEast *QuadTreeNode
	Divided   bool
}

type QuadTree struct {
	root *QuadTreeNode
	mu   sync.RWMutex
}

func NewQuadTree(minLat, maxLat, minLng, maxLng float64) *QuadTree {
	return &QuadTree{
		root: &QuadTreeNode{
			Boundary: BoundingBox{
				MinLat: minLat,
				MaxLat: maxLat,
				MinLng: minLng,
				MaxLng: maxLng,
			},
			Drivers: make([]*models.Driver, 0),
			Depth:   0,
			Divided: false,
		},
	}
}

func (qt *QuadTree) Insert(driver *models.Driver) bool {
	qt.mu.Lock()
	defer qt.mu.Unlock()
	return qt.root.insert(driver)
}

func (node *QuadTreeNode) insert(driver *models.Driver) bool {
	if !node.Boundary.Contains(driver.Location.Lat, driver.Location.Lng) {
		return false
	}

	if len(node.Drivers) < MaxCapacity && !node.Divided {
		node.Drivers = append(node.Drivers, driver)
		return true
	}

	if !node.Divided && node.Depth < MaxDepth {
		node.subdivide()
	}

	if node.Divided {
		if node.NorthWest.insert(driver) {
			return true
		}
		if node.NorthEast.insert(driver) {
			return true
		}
		if node.SouthWest.insert(driver) {
			return true
		}
		if node.SouthEast.insert(driver) {
			return true
		}
	}

	return false
}

func (node *QuadTreeNode) subdivide() {
	midLat := (node.Boundary.MinLat + node.Boundary.MaxLat) / 2
	midLng := (node.Boundary.MinLng + node.Boundary.MaxLng) / 2

	node.NorthWest = &QuadTreeNode{
		Boundary: BoundingBox{
			MinLat: midLat,
			MaxLat: node.Boundary.MaxLat,
			MinLng: node.Boundary.MinLng,
			MaxLng: midLng,
		},
		Drivers: make([]*models.Driver, 0),
		Depth:   node.Depth + 1,
	}

	node.NorthEast = &QuadTreeNode{
		Boundary: BoundingBox{
			MinLat: midLat,
			MaxLat: node.Boundary.MaxLat,
			MinLng: midLng,
			MaxLng: node.Boundary.MaxLng,
		},
		Drivers: make([]*models.Driver, 0),
		Depth:   node.Depth + 1,
	}

	node.SouthWest = &QuadTreeNode{
		Boundary: BoundingBox{
			MinLat: node.Boundary.MinLat,
			MaxLat: midLat,
			MinLng: node.Boundary.MinLng,
			MaxLng: midLng,
		},
		Drivers: make([]*models.Driver, 0),
		Depth:   node.Depth + 1,
	}

	node.SouthEast = &QuadTreeNode{
		Boundary: BoundingBox{
			MinLat: node.Boundary.MinLat,
			MaxLat: midLat,
			MinLng: midLng,
			MaxLng: node.Boundary.MaxLng,
		},
		Drivers: make([]*models.Driver, 0),
		Depth:   node.Depth + 1,
	}

	for _, driver := range node.Drivers {
		node.NorthWest.insert(driver)
		node.NorthEast.insert(driver)
		node.SouthWest.insert(driver)
		node.SouthEast.insert(driver)
	}

	node.Drivers = nil
	node.Divided = true
}

func (qt *QuadTree) SearchRadius(lat, lng, radiusKm float64) []*models.Driver {
	qt.mu.RLock()
	defer qt.mu.RUnlock()

	latDelta := radiusKm / 111.0
	lngDelta := radiusKm / (111.0 * math.Cos(lat*math.Pi/180))

	searchBox := BoundingBox{
		MinLat: lat - latDelta,
		MaxLat: lat + latDelta,
		MinLng: lng - lngDelta,
		MaxLng: lng + lngDelta,
	}

	results := make([]*models.Driver, 0)
	qt.root.searchInBoundary(&searchBox, &results)

	filtered := make([]*models.Driver, 0)
	for _, driver := range results {
		dist := Haversine(lat, lng, driver.Location.Lat, driver.Location.Lng)
		if dist <= radiusKm {
			filtered = append(filtered, driver)
		}
	}

	return filtered
}

func (node *QuadTreeNode) searchInBoundary(searchBox *BoundingBox, results *[]*models.Driver) {
	if !node.Boundary.Intersects(searchBox) {
		return
	}

	if !node.Divided {
		for _, driver := range node.Drivers {
			if searchBox.Contains(driver.Location.Lat, driver.Location.Lng) {
				*results = append(*results, driver)
			}
		}
		return
	}

	if node.NorthWest != nil {
		node.NorthWest.searchInBoundary(searchBox, results)
	}
	if node.NorthEast != nil {
		node.NorthEast.searchInBoundary(searchBox, results)
	}
	if node.SouthWest != nil {
		node.SouthWest.searchInBoundary(searchBox, results)
	}
	if node.SouthEast != nil {
		node.SouthEast.searchInBoundary(searchBox, results)
	}
}

func (qt *QuadTree) Remove(driverID string) bool {
	qt.mu.Lock()
	defer qt.mu.Unlock()
	return qt.root.remove(driverID)
}

func (node *QuadTreeNode) remove(driverID string) bool {
	if !node.Divided {
		for i, driver := range node.Drivers {
			if driver.ID == driverID {
				node.Drivers = append(node.Drivers[:i], node.Drivers[i+1:]...)
				return true
			}
		}
		return false
	}

	return node.NorthWest.remove(driverID) ||
		node.NorthEast.remove(driverID) ||
		node.SouthWest.remove(driverID) ||
		node.SouthEast.remove(driverID)
}

