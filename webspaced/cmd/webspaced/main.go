package main

import (
	"log"

	"github.com/netsoc/webspace-ng/webspaced/internal/server"
)

func main() {
	srv := server.NewServer()
	log.Fatal(srv.Start("/run/webspaced/server.sock"))
}
