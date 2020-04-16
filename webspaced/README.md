# Backend
This directory contains the backend portion of `webspace-ng`, which talks to LXD to manage containers, Traefik (via
Redis) and users (via an arbitrary frontend by means of a REST API).

REST API documentation can be found [here](api.md).

## Deploying
TL;DR: Edit `docker-compose.release.yaml` to your liking (removing the `build` sections) and start with `docker-compose up`.

 - You'll need to install [LXD](https://linuxcontainers.org/lxd/)
 - Edit `config.sample.yaml` to configure the daemon and set defaults for user containers
 - In order for `webspaced` to be able to see the names of users connecting over a Unix socket to the API, `pw_gr_proxy`
will need to be compiled via `gcc -o pw_gr_proxy pw_gr_proxy.c`. You can then start the proxy, passing the path for the
Unix socket on the host - make sure this is accessible by the `webspaced` container

## Developing
A Linux system is required.

[Docker](https://docs.docker.com/install/) and [Docker Compose](https://docs.docker.com/compose/install/) should be
installed.

You'll also need to install [LXD](https://linuxcontainers.org/lxd/getting-started-cli/). _Note: If you installed LXD via
Snap (e.g. on Ubuntu), you'll need to edit `docker-compose.yaml` and use the correct Unix socket path!_

Run `docker-compose up` to start the server - any changes to sources will automatically recompile and restart the
server.

You can then issue requests against the server, e.g. via `curl`:
`curl --unix-socket sockets/server.sock http://localhost`
