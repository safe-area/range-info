package main

import (
	"github.com/safe-area/range-info/internal/api"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	server := api.New(":8080")

	errChan := make(chan error, 1)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		errChan <- server.Start()
	}()

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
