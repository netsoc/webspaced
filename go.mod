module github.com/netsoc/webspaced

go 1.15

require (
	github.com/cenkalti/backoff/v4 v4.0.2
	github.com/devplayer0/http-swagger v0.0.0-20200916205217-5f599a45ac7b
	github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	github.com/flosch/pongo2 v0.0.0-20200913210552-0d938eb266f3 // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/githubnemo/CompileDaemon v1.2.1
	github.com/go-bindata/go-bindata/v3 v3.1.3
	github.com/go-redis/redis/v7 v7.4.0
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/jetstack/cert-manager v1.0.2
	github.com/juju/webbrowser v1.0.0 // indirect
	github.com/lxc/lxd v0.0.0-20201005111517-3f2b50ee46c9
	github.com/magiconair/properties v1.8.4 // indirect
	github.com/mitchellh/mapstructure v1.3.3
	github.com/netsoc/iam/client v1.0.10
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/rogpeppe/fastuuid v1.2.0 // indirect
	github.com/rs/cors v1.7.0
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/afero v1.4.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/traefik/traefik/v2 v2.3.1
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0 // indirect
	golang.org/x/net v0.0.0-20201002202402-0a1ea396d57c // indirect
	gopkg.in/httprequest.v1 v1.2.1 // indirect
	gopkg.in/ini.v1 v1.61.0 // indirect
	gopkg.in/macaroon-bakery.v2 v2.2.0 // indirect
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5 // indirect
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	k8s.io/utils v0.0.0-20200912215256-4140de9c8800 // indirect
)

// Netsoc Traefik
replace github.com/traefik/traefik/v2 => github.com/netsoc/traefik/v2 v2.3.2-0.20201005105929-66099221d9de

// Containous forks
replace (
	github.com/abbot/go-http-auth => github.com/containous/go-http-auth v0.4.1-0.20200324110947-a37a7636d23e
	github.com/go-check/check => github.com/containous/check v0.0.0-20170915194414-ca0bf163426a
	github.com/mailgun/minheap => github.com/containous/minheap v0.0.0-20190809180810-6e71eb837595
	github.com/mailgun/multibuf => github.com/containous/multibuf v0.0.0-20190809014333-8b6c9a7e6bba
)

// Docker v19.03.6
replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20200204220554-5f6d6f3f2203
