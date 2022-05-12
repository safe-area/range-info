package service

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/safe-area/range-info/config"
	"github.com/safe-area/range-info/internal/models"
	"github.com/safe-area/range-info/internal/nats_provider"
	"github.com/sirupsen/logrus"
	"github.com/uber/h3-go"
)

const (
	shardTemplate = "GET_DATA_SHARD_"
	defaultShard  = "GET_DATA_SHARD_DEFAULT"
)

type Service interface {
	GetData(ixs []h3.H3Index) (map[h3.H3Index]models.HexData, error)
	Prepare()
}

type service struct {
	nats   *nats_provider.NATSProvider
	shards map[int]string
	cfg    *config.Config
}

func NewService(cfg *config.Config, provider *nats_provider.NATSProvider) Service {
	return &service{
		nats:   provider,
		shards: make(map[int]string),
		cfg:    cfg,
	}
}

func (s *service) Prepare() {
	for _, v := range s.cfg.Shards {
		s.shards[v] = fmt.Sprint(shardTemplate, v)
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

	// TODO SHARDS
	subj := defaultShard
	msg, err := s.nats.Request(subj, storeReqBody)
	if err != nil {
		logrus.Errorf("sendToShard: nats request error: %s", err)
		return nil, err
	}
	resp := make(map[h3.H3Index]models.HexData)
	if err = jsoniter.Unmarshal(msg.Data, &resp); err != nil {
		logrus.Errorf("getDataDev: error while unmarshalling response: %s", err)
		return nil, err
	}
	return resp, nil
}

func (s *service) getSubj(hex int) string {
	if v, ok := s.shards[hex]; ok {
		return v
	} else {
		return defaultShard
	}
}
