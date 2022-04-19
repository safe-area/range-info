package api

import (
	"encoding/json"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/lab259/cors"
	"github.com/safe-area/range-info/config"
	"github.com/safe-area/range-info/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/uber/h3-go"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttprouter"
	"math/rand"
	"net/http"
	"time"
)

type Server struct {
	r          *fasthttprouter.Router
	serv       *fasthttp.Server
	httpClient *fasthttp.Client
	cfg        *config.Config
}

func New(cfg *config.Config) *Server {
	innerRouter := fasthttprouter.New()
	innerHandler := innerRouter.Handler
	s := &Server{
		innerRouter,
		&fasthttp.Server{
			ReadTimeout:  time.Duration(600) * time.Second,
			WriteTimeout: time.Duration(600) * time.Second,
			IdleTimeout:  time.Duration(600) * time.Second,
			Handler:      cors.AllowAll().Handler(innerHandler),
		},
		new(fasthttp.Client),
		cfg,
	}

	s.r.POST("/api/v1/range", s.RangeHandler)
	s.r.POST("/api/v1/area", s.AreaHandler)
	//s.r.POST("/api/v1/trace", s.TraceHandler)

	return s
}

func (s *Server) RangeHandler(ctx *fasthttp.RequestCtx, ps fasthttprouter.Params) {
	body := ctx.PostBody()
	var geoData models.GeoData
	err := jsoniter.Unmarshal(body, &geoData)
	if err != nil {
		logrus.Errorf("RangeHandler: error while unmarshalling request: %s", err)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	storeReq := make(map[h3.H3Index]models.HexData)
	var storeReqBody []byte
	storeReqBody, err = json.Marshal(storeReq)
	if err != nil {
		logrus.Error("RangeHandler: marshal error:", err)
		return
	}

	httpReq := fasthttp.AcquireRequest()
	httpResp := fasthttp.AcquireResponse()
	httpReq.Header.SetMethod("GET")
	httpReq.Header.SetContentType("application/json")
	httpReq.SetRequestURI(s.cfg.Storage.Host + "/api/v1/get")
	httpReq.SetBody(storeReqBody)
	if err := s.httpClient.Do(httpReq, httpResp); err != nil {
		logrus.Error("RangeHandler: Do request error", err)
		return
	}
	if httpResp.StatusCode() != http.StatusOK {
		logrus.Error("RangeHandler: status code:", httpResp.StatusCode())
		return
	}
	fasthttp.ReleaseRequest(httpReq)
	fasthttp.ReleaseResponse(httpResp)

	index := h3.FromGeo(h3.GeoCoord{Latitude: geoData.Latitude, Longitude: geoData.Longitude}, models.HexResolution)
	fmt.Printf("\033[0;33mTimestamp: \033[4;35m\"%v\"\033[0;33m, Coordinats: \033[4;35m(%v,%v)\033[0;33m, HexIndex(res: 12): \033[4;35m%v(%v)\033[0m\n", time.Unix(geoData.Timestamp, 0),
		geoData.Latitude, geoData.Longitude, index, h3.BaseCell(index))
	ring := h3.KRing(index, 2)
	hexMap := make(map[h3.H3Index]models.HexProperties, len(ring))
	for _, v := range ring {
		hexMap[v] = models.HexProperties{
			Healthy:    rand.Intn(30),
			Suspicious: rand.Intn(4),
			Infected:   rand.Intn(2),
		}
	}
	gj, err := getGeoJson(hexMap)
	if err != nil {
		logrus.Errorf("TestHandler: error while marshalling geojson: %s", err)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.Write(gj)
}

func (s *Server) AreaHandler(ctx *fasthttp.RequestCtx, ps fasthttprouter.Params) {
	body := ctx.PostBody()
	var area models.Area
	err := jsoniter.Unmarshal(body, &area)
	if err != nil {
		logrus.Errorf("TestHandler: error while unmarshalling request: %s", err)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}
	polygon := h3.GeoPolygon{}
	for _, p := range area.Polygon {
		polygon.Geofence = append(polygon.Geofence, h3.GeoCoord{
			Latitude: p.Latitude, Longitude: p.Longitude,
		})
	}
	indexes := h3.Polyfill(polygon, models.HexResolution)
	hexMap := make(map[h3.H3Index]models.HexProperties, len(indexes))
	for _, v := range indexes {
		hexMap[v] = models.HexProperties{
			Healthy:    rand.Intn(30),
			Suspicious: rand.Intn(4),
			Infected:   rand.Intn(2),
		}
	}
	gj, err := getGeoJson(hexMap)
	if err != nil {
		logrus.Errorf("TestHandler: error while marshalling geojson: %s", err)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.Write(gj)
}

func (s *Server) Start() error {
	return fmt.Errorf("server start: %s", s.serv.ListenAndServe(s.cfg.Port))
}
func (s *Server) Shutdown() error {
	return s.serv.Shutdown()
}

func getGeoJson(m map[h3.H3Index]models.HexProperties) ([]byte, error) {
	features := make([]map[string]interface{}, 0, len(m))
	for k, v := range m {
		gb := h3.ToGeoBoundary(k)
		gbSlices := make([][][]float64, 1)
		for _, g := range gb {
			gbSlices[0] = append(gbSlices[0], []float64{g.Longitude, g.Latitude})
		}
		gbSlices[0] = append(gbSlices[0], []float64{gb[0].Longitude, gb[0].Latitude})
		feature := make(map[string]interface{})
		feature["type"] = "Feature"
		feature["geometry"] = map[string]interface{}{
			"type":        "Polygon",
			"coordinates": gbSlices,
		}
		var color string
		var opacity float64
		if v.Infected != 0 {
			color = "red"
			opacity = 4 * float64(v.Infected) / float64(v.Healthy+v.Suspicious+v.Infected)
			if opacity > 0.8 {
				opacity = 0.8
			}
		} else if v.Suspicious != 0 {
			color = "yellow"
			opacity = 2 * float64(v.Suspicious) / float64(v.Healthy+v.Suspicious+v.Infected)
			if opacity > 0.8 {
				opacity = 0.8
			}
		} else if v.Healthy != 0 {
			color = "green"
			opacity = 0.5
		} else {
			color = "white"
			opacity = 0.0
		}
		feature["properties"] = map[string]interface{}{
			"healthy":      v.Healthy,
			"suspicious":   v.Suspicious,
			"infected":     v.Infected,
			"fill":         color,
			"stroke-width": "1",
			"fill-opacity": opacity,
		}
		//feature["style"] = map[string]interface{}{
		//	"fill":         color,
		//	"stroke-width": "1",
		//	"fill-opacity": opacity,
		//}
		features = append(features, feature)
	}
	geoJsonStruct := map[string]interface{}{
		"type":     "FeatureCollection",
		"features": features,
	}
	return jsoniter.Marshal(geoJsonStruct)
}
