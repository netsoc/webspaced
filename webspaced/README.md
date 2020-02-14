# Backend
This repository contains the backend portion of `webspace-ng`, which talks to LXD to manage containers.

REST API documentation can be found [here](api.md).

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
