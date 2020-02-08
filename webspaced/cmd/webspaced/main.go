package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/netsoc/webspace-ng/webspaced/internal/server"
	"golang.org/x/sys/unix"
)

func main() {
	srv := server.NewServer()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, unix.SIGINT, unix.SIGTERM)

	go func() {
		<-sigs
		srv.Stop()
	}()

	log.Fatal(srv.Start("/run/webspaced/server.sock"))
}
