module github.com/netsoc/webspaced

go 1.16

require (
	github.com/cenkalti/backoff/v4 v4.1.1
	github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	github.com/flosch/pongo2 v0.0.0-20200913210552-0d938eb266f3 // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/githubnemo/CompileDaemon v1.3.0
	github.com/go-bindata/go-bindata/v3 v3.1.3
	github.com/go-openapi/runtime v0.19.29
	github.com/go-openapi/spec v0.20.3 // indirect
	github.com/go-redis/redis/v7 v7.4.1
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/jetstack/cert-manager v1.4.0
	github.com/juju/webbrowser v1.0.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/lxc/lxd v0.0.0-20210721222701-a124a46b7614
	github.com/mitchellh/mapstructure v1.4.1
	github.com/netsoc/iam/client v1.0.11
	github.com/rs/cors v1.8.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.8.1
	github.com/traefik/traefik/v2 v2.5.0-rc2
	golang.org/x/net v0.0.0-20210716203947-853a461950ff // indirect
	golang.org/x/tools v0.1.5 // indirect
	gopkg.in/httprequest.v1 v1.2.1 // indirect
	gopkg.in/macaroon-bakery.v2 v2.3.0 // indirect
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5 // indirect
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v0.21.3
)

// Netsoc Traefik
replace github.com/traefik/traefik/v2 => github.com/netsoc/traefik/v2 v2.5.0-rc2-netsoc

// Containous forks
replace (
	github.com/abbot/go-http-auth => github.com/containous/go-http-auth v0.4.1-0.20200324110947-a37a7636d23e
	github.com/go-check/check => github.com/containous/check v0.0.0-20170915194414-ca0bf163426a
	github.com/mailgun/minheap => github.com/containous/minheap v0.0.0-20190809180810-6e71eb837595
	github.com/mailgun/multibuf => github.com/containous/multibuf v0.0.0-20190809014333-8b6c9a7e6bba
)
