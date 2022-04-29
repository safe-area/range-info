package models

const HexResolution = 11

type GeoData struct {
	Latitude  float64 `json:"lt"`
	Longitude float64 `json:"lg"`
	Timestamp int64   `json:"ts"`
}

type Area struct {
	Polygon   []Point `json:"polygon"`
	Timestamp int64   `json:"ts"`
}

type Point struct {
	Latitude  float64 `json:"lt"`
	Longitude float64 `json:"lg"`
}
