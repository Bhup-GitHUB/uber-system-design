package router

import (
	"fmt"
	"sync"
)

type City struct {
	Name   string
	MinLat float64
	MaxLat float64
	MinLng float64
	MaxLng float64
}

type GeoRouter struct {
	cities map[string]*City
	mu     sync.RWMutex
}

func NewGeoRouter() *GeoRouter {
	return &GeoRouter{
		cities: make(map[string]*City),
	}
}

func (gr *GeoRouter) RegisterCity(name string, minLat, maxLat, minLng, maxLng float64) {
	gr.mu.Lock()
	defer gr.mu.Unlock()
	gr.cities[name] = &City{
		Name:   name,
		MinLat: minLat,
		MaxLat: maxLat,
		MinLng: minLng,
		MaxLng: maxLng,
	}
}

func (gr *GeoRouter) GetCity(lat, lng float64) (string, error) {
	gr.mu.RLock()
	defer gr.mu.RUnlock()

	for name, city := range gr.cities {
		if lat >= city.MinLat && lat <= city.MaxLat &&
			lng >= city.MinLng && lng <= city.MaxLng {
			return name, nil
		}
	}

	return "", fmt.Errorf("no city found for location: %f, %f", lat, lng)
}

func (gr *GeoRouter) ListCities() []string {
	gr.mu.RLock()
	defer gr.mu.RUnlock()

	cities := make([]string, 0, len(gr.cities))
	for name := range gr.cities {
		cities = append(cities, name)
	}
	return cities
}

