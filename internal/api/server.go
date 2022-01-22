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
	gj, err := getGeoJson(map[h3.H3Index]models.HexProperties{
		index: {
			Healthy:    20,
			Suspicious: 3,
			Infected:   1,
		},
	})
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
			gbSlices[0] = append(gbSlices[0], []float64{g.Latitude, g.Longitude})
		}
		feature := make(map[string]interface{})
		feature["type"] = "Feature"
		feature["geometry"] = map[string]interface{}{
			"type":        "Polygon",
			"coordinates": gbSlices,
		}
		feature["properties"] = v
		features = append(features, feature)
	}
	geoJsonStruct := map[string]interface{}{
		"type":     "FeatureCollection",
		"features": features,
	}
	return jsoniter.Marshal(geoJsonStruct)
}
