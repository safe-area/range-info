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
		logrus.Warnf("HandleLastStatusOfRids error while unmarshalling request: %s", err)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}
	index := h3.FromGeo(h3.GeoCoord{Latitude: geoData.Latitude, Longitude: geoData.Longitude}, 12)
	logrus.Infof("Timestamp: %v, Coordinats: (%v,%v), HexIndex(res: 12): %v", time.Unix(geoData.Timestamp, 0),
		geoData.Latitude, geoData.Longitude, index)
}

func (s *Server) Start() error {
	return fmt.Errorf("server start: %s", s.serv.ListenAndServe(s.port))
}
func (s *Server) Shutdown() error {
	return s.serv.Shutdown()
}
