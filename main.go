package main

import (
	"github.com/safe-area/range-info/config"
	"github.com/safe-area/range-info/internal/api"
	"github.com/safe-area/range-info/internal/nats_provider"
	"github.com/safe-area/range-info/internal/service"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg, err := config.ParseConfig("./config/config.json")
	if err != nil {
		logrus.Fatalf("parse config error: %v", err)
	}

	provider := nats_provider.New(cfg.NATS.URLs)
	err = provider.Open()
	if err != nil {
		logrus.Fatalf("open nats conn error: %v", err)
	}

	svc := service.NewService(cfg, provider)
	svc.Prepare()

	server := api.New(cfg, svc)

	errChan := make(chan error, 1)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		errChan <- server.Start()
	}()

	logrus.Info("server started")
	select {
	case err := <-errChan:
		if err != nil {
			logrus.Errorf("server crushed with error: %v", err)
		}
		server.Shutdown()
	case <-signalChan:
		logrus.Info("received a signal, shutting down...")
		server.Shutdown()
	}
}
