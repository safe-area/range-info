package models

type GeoData struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp int64   `json:"time"`
}

type HexProperties struct {
	Healthy    int `json:"healthy"`
	Suspicious int `json:"suspicious"`
	Infected   int `json:"infected"`
}
