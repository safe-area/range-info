package models

import "github.com/uber/h3-go"

type GetRequest struct {
	Indexes []h3.H3Index `json:"indexes"`
}
