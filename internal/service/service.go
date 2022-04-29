package service

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/safe-area/range-info/config"
	"github.com/safe-area/range-info/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/uber/h3-go"
	"github.com/valyala/fasthttp"
	"net/http"
)

type Service interface {
	GetData(ixs []h3.H3Index) (map[h3.H3Index]models.HexData, error)
}

type service struct {
	httpClient *fasthttp.Client
	cfg        *config.Config
}

func NewService(cfg *config.Config) Service {
	return &service{
		httpClient: new(fasthttp.Client),
		cfg:        cfg,
	}
}

func (s *service) GetData(ixs []h3.H3Index) (map[h3.H3Index]models.HexData, error) {
	if s.cfg.Dev {
		return s.getDataDev(ixs)
	} else {
		return s.getData(ixs)
	}
}

func (s *service) getData(ixs []h3.H3Index) (map[h3.H3Index]models.HexData, error) {
	resp := make(map[h3.H3Index]models.HexData)
	return resp, nil
}

func (s *service) getDataDev(ixs []h3.H3Index) (map[h3.H3Index]models.HexData, error) {
	var (
		err          error
		storeReqBody []byte
	)

	storeReq := models.GetRequest{
		Indexes: ixs,
	}
	storeReqBody, err = jsoniter.Marshal(storeReq)
	if err != nil {
		logrus.Errorf("getDataDev: error while marshalling request: %s", err)
		return nil, err
	}

	httpReq := fasthttp.AcquireRequest()
	httpResp := fasthttp.AcquireResponse()
	httpReq.Header.SetMethod("GET")
	httpReq.Header.SetContentType("application/json")
	httpReq.SetRequestURI(s.cfg.Storage.Host + "/api/v1/get")
	httpReq.SetBody(storeReqBody)
	if err = s.httpClient.Do(httpReq, httpResp); err != nil {
		logrus.Error("getDataDev: Do request error", err)
		fasthttp.ReleaseRequest(httpReq)
		fasthttp.ReleaseResponse(httpResp)
		return nil, err
	}
	if httpResp.StatusCode() != http.StatusOK {
		logrus.Error("getDataDev: status code:", httpResp.StatusCode())
		fasthttp.ReleaseRequest(httpReq)
		fasthttp.ReleaseResponse(httpResp)
		return nil, err
	}
	resp := make(map[h3.H3Index]models.HexData)
	if err = jsoniter.Unmarshal(httpResp.Body(), resp); err != nil {
		logrus.Errorf("getDataDev: error while unmarshalling response: %s", err)
		fasthttp.ReleaseRequest(httpReq)
		fasthttp.ReleaseResponse(httpResp)
		return nil, err
	}
	fasthttp.ReleaseRequest(httpReq)
	fasthttp.ReleaseResponse(httpResp)
	return resp, nil
}
