FROM golang:1.15-alpine AS builder
RUN apk --no-cache add git gcc musl-dev linux-headers

WORKDIR /usr/local/lib/webspaced
COPY go.* ./
RUN go mod download

COPY tools.go ./
RUN cat tools.go | sed -nr 's|^\t_ "(.+)"$|\1|p' | xargs -tI % go get %

COPY static/ ./static/
COPY cmd/ ./cmd/
COPY pkg/ ./pkg/
COPY internal/ ./internal/
RUN go-bindata -fs -o internal/data/bindata.go -pkg data -prefix static/ static/...
RUN mkdir bin/ && go build -ldflags '-s -w' -o bin/ ./cmd/...


FROM alpine:3.12

COPY --from=builder /usr/local/lib/webspaced/bin/* /usr/local/bin/

EXPOSE 80/tcp
ENTRYPOINT ["/usr/local/bin/webspaced"]

LABEL org.opencontainers.image.source https://github.com/netsoc/webspaced
