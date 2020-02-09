# Backend
This repository contains the backend portion of `webspace-ng`, which talks to LXD to manage containers.

## Developing
[Docker](https://docs.docker.com/install/) and [Docker Compose](https://docs.docker.com/compose/install/) should be
installed.

Run `docker-compose up` to start the server - any changes to sources will automatically recompile and restart the
server.

You can then issue requests against the server, e.g. via `curl`:
`curl --unix-socket sockets/server.sock http://localhost`
