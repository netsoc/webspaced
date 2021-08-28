# Introduction

webspaced is essentially the "glue" that allows webspaces to work. [kubelan](https://github.com/devplayer0/kubelan) is
used to network all of the components and containers together.

## Features

### REST API

webspaced provides a REST  API that allows for users to manage their container. This is effectively a high-level proxy
to the LXD API wrapped with per-user authentication provided by [iamd](../../../iam/). The API is well-documented via
an OpenAPI spec at `static/api.yaml`.

!!! tip
    The API can be browsed and tested at [webspaced.netsoc.ie/swagger](https://webspaced.netsoc.ie/swagger).

### Traefik config generation

In order to route HTTP(S) traffic to containers, a HTTP reverse proxy is needed. Traefik is used for its flexibility in
dynamic configuration. This is based on the state of containers and events delivered by LXD when state changes.
Currently Kubernetes (`IngressRoute` and `IngressRouteTCP` custom resources) and Redis backends are supported
for discovery, although the Redis backend is currently broken. Traefik's new plugin system allows for new config
providers to be integrated intro Traefik with relative ease, so this might be implemented in the future.

When a container is running, a configuration with the container's current IP address will be generated. If
it's not running, a configuration with Netsoc's custom `webspaceBoot` middleware will be used. This causes Traefik to
ask webspaced to boot the webspace and wait for it to get an IP address, giving Heroku or AWS Lambda-style cold boot
behaviour.

Automatic TLS is supported for custom domains, with cert-manager `Certificate` objects created as necessary
for custom domains. It's also possible to use custom TLS certs (and effectively disable TLS termination).

### Port forwarding

Although Traefik provides TCP proxying functionality, it's limited in that the actual frontend ports cannot be
configured dynamically. Instead, webspaced does its own TCP proxying. When a connection comes through for a webspace
that isn't running, the connection is held open until the webspace has started.

If webspaced is deployed in Kubernetes, it can be configured to update a `Service` with the ports that should be
exposed.

## Deployment

A Helm chart is provided, see [our charts repo](https://github.com/netsoc/charts). There are quite a number of
configuration options, see `config.sample.yaml` for a complete list. LXD doesn't necessarily need to be deployed inside
Kubernetes, but since webspaced and other components need to be able to talk to containers directly,
[lxd8s](https://github.com/devplayer/lxd8s) provides a Kubernetes-based solution which integrates with kubelan.

You'll also need to generate a client certificate for webspaced to authenticate with against LXD. Any self-signed
certificate will do, `lxc config trust add my-cert.crt` should be used to import the certificate into an LXD
installation.

!!! note
    webspaced is currently deployed as a singleton `StatefulSet`. Although webspaced doesn't store any state itself, a
    `StatefulSet` is used due to the potential for clashes with Kubernetes resources being modified by multiple
    instances of the application running at once. `StatefulSet` ensures that only one pod for each replica is running at
    any given time.
