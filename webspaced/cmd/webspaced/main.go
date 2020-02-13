package main

import (
	"os"
	"os/signal"

	"github.com/netsoc/webspace-ng/webspaced/internal/server"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func init() {
	if val := os.Getenv("DEBUG"); val != "" {
		log.SetLevel(log.DebugLevel)
	}
}
func main() {
	srv := server.NewServer()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, unix.SIGINT, unix.SIGTERM)

	go func() {
		log.Info("Starting server...")
		if err := srv.Start("/run/webspaced/server.sock"); err != nil {
			log.WithField("error", err).Fatal("Failed to start server")
		}
	}()

	<-sigs
	srv.Stop()
}
