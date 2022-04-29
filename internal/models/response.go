package models

type Response struct {
	Hexes []HexResponse `json:"hs"`
}

type HexResponse struct {
	Boundaries []Point `json:"bs"`
	Healthy    int     `json:"healthy"`
	Infected   int     `json:"infected"`
}
