#!/bin/sh
set -e

go-bindata -debug -fs -o internal/data/bindata.go -pkg data -prefix static/ static/...
go build -o bin/webspaced ./cmd/webspaced
