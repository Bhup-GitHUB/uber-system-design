package geospatial

type BoundingBox struct {
	MinLat float64
	MaxLat float64
	MinLng float64
	MaxLng float64
}

func (bb *BoundingBox) Contains(lat, lng float64) bool {
	return lat >= bb.MinLat && lat <= bb.MaxLat &&
		lng >= bb.MinLng && lng <= bb.MaxLng
}

func (bb *BoundingBox) Intersects(other *BoundingBox) bool {
	return !(other.MinLat > bb.MaxLat ||
		other.MaxLat < bb.MinLat ||
		other.MinLng > bb.MaxLng ||
		other.MaxLng < bb.MinLng)
}

