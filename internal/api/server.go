package api

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/lab259/cors"
	"github.com/safe-area/range-info/config"
	"github.com/safe-area/range-info/internal/models"
	"github.com/safe-area/range-info/internal/service"
	"github.com/sirupsen/logrus"
	"github.com/uber/h3-go"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttprouter"
	"time"
)

type Server struct {
	r    *fasthttprouter.Router
	serv *fasthttp.Server
	svc  service.Service
	cfg  *config.Config
}

func New(cfg *config.Config, svc service.Service) *Server {
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
		svc,
		cfg,
	}

	s.r.POST("/api/v1/range", s.RangeHandler)
	s.r.POST("/api/v1/area", s.AreaHandler)
	//s.r.POST("/api/v1/trace", s.TraceHandler)

	return s
}

func (s *Server) RangeHandler(ctx *fasthttp.RequestCtx, ps fasthttprouter.Params) {
	var (
		geoData models.GeoData
		hexMap  map[h3.H3Index]models.HexData
		err     error
		respBs  []byte
	)
	err = jsoniter.Unmarshal(ctx.PostBody(), &geoData)
	if err != nil {
		logrus.Error("RangeHandler: error while unmarshalling request: ", err)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	index := h3.FromGeo(h3.GeoCoord{Latitude: geoData.Latitude, Longitude: geoData.Longitude}, models.HexResolution)
	fmt.Printf("\033[0;33mTimestamp: \033[4;35m\"%v\"\033[0;33m, Coordinats: \033[4;35m(%v,%v)\033[0;33m, HexIndex(res: 12): \033[4;35m%v(%v)\033[0m\n", time.Unix(geoData.Timestamp, 0),
		geoData.Latitude, geoData.Longitude, index, h3.BaseCell(index))
	ring := h3.KRing(index, 2)

	hexMap, err = s.svc.GetData(ring)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}
	resp := ConvertToResponse(hexMap)
	respBs, err = jsoniter.Marshal(resp)
	if err != nil {
		logrus.Errorf("RangeHandler: error while marshalling geojson: %s", err)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, err = ctx.Write(respBs)
	if err != nil {
		logrus.Errorf("RangeHandler: write response error: %s", err)
	}
}

func (s *Server) AreaHandler(ctx *fasthttp.RequestCtx, ps fasthttprouter.Params) {
	var (
		area   models.Area
		hexMap map[h3.H3Index]models.HexData
		err    error
		respBs []byte
	)
	err = jsoniter.Unmarshal(ctx.PostBody(), &area)
	if err != nil {
		logrus.Errorf("AreaHandler: error while unmarshalling request: %s", err)
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
	hexMap, err = s.svc.GetData(indexes)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}
	resp := ConvertToResponse(hexMap)
	respBs, err = jsoniter.Marshal(resp)
	if err != nil {
		logrus.Errorf("AreaHandler: error while marshalling geojson: %s", err)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, err = ctx.Write(respBs)
	if err != nil {
		logrus.Errorf("AreaHandler: write response error: %s", err)
	}
}

func (s *Server) Start() error {
	return fmt.Errorf("server start: %s", s.serv.ListenAndServe(s.cfg.Port))
}
func (s *Server) Shutdown() error {
	return s.serv.Shutdown()
}

func ConvertToResponse(m map[h3.H3Index]models.HexData) models.Response {
	resp := models.Response{}
	for k, v := range m {
		gb := h3.ToGeoBoundary(k)
		hbs := make([]models.Point, 6)
		for i, g := range gb {
			hbs[i] = models.Point{Latitude: g.Latitude, Longitude: g.Longitude}
		}
		resp.Hexes = append(resp.Hexes, models.HexResponse{
			Boundaries: hbs,
			Healthy:    v.Healthy,
			Infected:   v.Infected,
		})
	}
	return resp
}

//func getGeoJson(m map[h3.H3Index]models.HexProperties) ([]byte, error) {
//	features := make([]map[string]interface{}, 0, len(m))
//	for k, v := range m {
//		gb := h3.ToGeoBoundary(k)
//		gbSlices := make([][][]float64, 1)
//		for _, g := range gb {
//			gbSlices[0] = append(gbSlices[0], []float64{g.Longitude, g.Latitude})
//		}
//		gbSlices[0] = append(gbSlices[0], []float64{gb[0].Longitude, gb[0].Latitude})
//		feature := make(map[string]interface{})
//		feature["type"] = "Feature"
//		feature["geometry"] = map[string]interface{}{
//			"type":        "Polygon",
//			"coordinates": gbSlices,
//		}
//		var color string
//		var opacity float64
//		if v.Infected != 0 {
//			color = "red"
//			opacity = 4 * float64(v.Infected) / float64(v.Healthy+v.Suspicious+v.Infected)
//			if opacity > 0.8 {
//				opacity = 0.8
//			}
//		} else if v.Suspicious != 0 {
//			color = "yellow"
//			opacity = 2 * float64(v.Suspicious) / float64(v.Healthy+v.Suspicious+v.Infected)
//			if opacity > 0.8 {
//				opacity = 0.8
//			}
//		} else if v.Healthy != 0 {
//			color = "green"
//			opacity = 0.5
//		} else {
//			color = "white"
//			opacity = 0.0
//		}
//		feature["properties"] = map[string]interface{}{
//			"healthy":      v.Healthy,
//			"suspicious":   v.Suspicious,
//			"infected":     v.Infected,
//			"fill":         color,
//			"stroke-width": "1",
//			"fill-opacity": opacity,
//		}
//		//feature["style"] = map[string]interface{}{
//		//	"fill":         color,
//		//	"stroke-width": "1",
//		//	"fill-opacity": opacity,
//		//}
//		features = append(features, feature)
//	}
//	geoJsonStruct := map[string]interface{}{
//		"type":     "FeatureCollection",
//		"features": features,
//	}
//	return jsoniter.Marshal(geoJsonStruct)
//}
