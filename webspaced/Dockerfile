FROM golang:1.13-alpine as builder

WORKDIR /usr/local/src
COPY go.* ./
RUN go mod download

COPY . .
RUN mkdir bin/ && go build -o bin/ ./cmd/...


FROM alpine:3.11

COPY --from=builder /usr/local/src/bin/* /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/webspaced"]
