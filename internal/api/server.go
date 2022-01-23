package api

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/lab259/cors"
	"github.com/safe-area/range-info/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/uber/h3-go"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttprouter"
	"math/rand"
	"time"
)

type Server struct {
	r    *fasthttprouter.Router
	serv *fasthttp.Server
	port string
}

func New(port string) *Server {
	innerRouter := fasthttprouter.New()
	innerHandler := innerRouter.Handler
	s := &Server{
		innerRouter,
		&fasthttp.Server{
			ReadTimeout:  time.Duration(5) * time.Second,
			WriteTimeout: time.Duration(5) * time.Second,
			IdleTimeout:  time.Duration(5) * time.Second,
			Handler:      cors.AllowAll().Handler(innerHandler),
		},
		port,
	}

	s.r.POST("/api/v1/test", s.TestHandler)

	return s
}

func (s *Server) TestHandler(ctx *fasthttp.RequestCtx, ps fasthttprouter.Params) {
	body := ctx.PostBody()
	var geoData models.GeoData
	err := jsoniter.Unmarshal(body, &geoData)
	if err != nil {
		logrus.Errorf("TestHandler: error while unmarshalling request: %s", err)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}
	index := h3.FromGeo(h3.GeoCoord{Latitude: geoData.Latitude, Longitude: geoData.Longitude}, 12)
	fmt.Printf("\033[0;33mTimestamp: \033[4;35m\"%v\"\033[0;33m, Coordinats: \033[4;35m(%v,%v)\033[0;33m, HexIndex(res: 12): \033[4;35m%v\033[0m\n", time.Unix(geoData.Timestamp, 0),
		geoData.Latitude, geoData.Longitude, index)
	ring := h3.KRing(index, 4)
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

func (s *Server) Start() error {
	return fmt.Errorf("server start: %s", s.serv.ListenAndServe(s.port))
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
