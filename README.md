# webspace-ng
`webspace-ng` is a project to provide containerised webspaces to Netsoc's members. Webspaces are a classic service
provided to members of large institutions such as universities giving an easy way to host their own website for free.
Typically, users are given a subdirectory such as `https://myuni.edu/~osullj19/` and can place simple HTML pages and
PHP or other CGI scripts in a designated place in their home directory.

This simple form of hosting was invaluable in the early days of the web when setting up a website was difficult and
expensive. In the days of services like GitHub pages offering a free and easy to use static hosting platform however,
the need to innovate is apparent!

Netsoc came up with idea of providing containerised webspaces for their members. Instead of being limited to HTML and
maybe PHP and MySQL, users would have full root access to their container and be able to choose their server operating
system and develop their site using any framework they would like. Using containers instead of virtual machines, a
reverse proxy, port forwarding and Heroku-style transparent booting make it possible to make these very flexible
webspaces available on the limited hardware of a university society such as Netsoc.

## Features
`webspace-ng` provides at its core a backend ([`webspaced`](webspaced/)) which communicates with the containerisation framework
([LXD](https://linuxcontainers.org/lxd/)), reverse proxy ([Traefik](https://traefik.io/traefik/)) and exposes a REST
API for frontends such as CLI's or web UI's.

 - CLI is a port of Netsoc's original version updated to communicate with the
[REST API exposed by the backend](webspaced/api.md) and can be found in the [`cli`](cli/) directory
