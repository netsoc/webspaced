# Environment

## Resources

Each container has the following resources (subject to increase when we get more
servers!):

- 1 vCPU
- 1GiB RAM
- 8GiB storage

## Boot policy

Due to our somewhat constrained resources, Netsoc has a policy for shutting down
long-running containers. If the node hosting your webspace is running low
on memory, _the longest running webspace will be shut down to reclaim
resources._

But fear not! `webspaced` has been designed to make this completely transparent.
If any endpoint of your webspace receives a connection (e.g. over HTTP at
https://myusername.ng.netsoc.ie and any custom domains you have, or through a
port forward for SSH or some other service), `webspaced` will delay the
connection until your webspace is ready.

This delay is made up of:

- The time it takes LXD to mark the container as "running" and
- A pre-configured delay to allow your applications to start up

The pre-configured delay can be
[configured](/cli/reference/netsoc_webspace_config_set/).

!!! warning
    Although the boot up process is transparent and should only result in the
    occasional delay if your webspace is not running, you must set up your
    applications to run on boot inside your webspace! This usually means
    creating a systemd service (if you're not using a server that comes
    pre-configured to run on boot, under Ubuntu generally any new package installed
    which provides a service will be automatically set to run on boot).

## Containers?

As mentioned in [the introduction](../), a webspace is a
VM-style container powered by [LXD](https://linuxcontainers.org/lxd/).
Containers generally refer to something like [Docker](https://www.docker.com/),
but Docker provides _application_ containers.

- A `Dockerfile` is used to build up the
system inside the container, and it is rarely logged into
- The files inside are
generally _ephemeral_ (destroyed on shutdown), with the exception of explicitly
defined volumes
- Generally the only process running inside the container the application itself

LXD provides _system_ containers, which although uses an isolation technology
very similar to Docker (and other application container frameworks), operates
somewhat differently

- Each container's storage is _persistent_, which means changes are saved on
shutdown.
- A full init system (like [systemd](https://systemd.io/)) runs inside, handling
system tasks like keeping track of service logs and running periodic jobs
