# REST API
`webspaced` exposes a REST API for managing webspace containers.

# Authentication
Since the API is exposed over a Unix socket, it is possible to authenticate a client by obtaining their UID via
`SO_PEERCRED`. If the user is root (UID 0) or a member of the `webspace-admin` group, the desired user may specified
in the `X-Webspace-User` header.

If the API is to be exposed over TCP in future, an additional authentication mechanism would be added.

# Endpoints
## `/v1/webspace`
### POST
Initialize the user's webspace.

Request:

 - `image`: String representing LXD image by alias or fingerprint
 - `password`: _(Optional)_ Root password
 - `sshKey`: _(Optional)_ SSH public key to add for `root` - specifying this will ensure SSH is installed / enabled and
    add a forward for port 22

Response:

Either HTTP 204 (No Content) and an empty response body (if `sshKey` was not present in the request) or HTTP 201
(Created) and the following:
 - `sshPort`: The external port for SSH access

### DELETE
Destroy the user's webspace.

Response:

HTTP 204 (No Content).


## `/v1/webspace/state`
### POST
Start the webspace container.

Response:

HTTP 204 (No Content).

### PUT
Reboot the webspace container.

Response:

HTTP 204 (No Content).

### DELETE
Shutdown the webspace container.

Response:

HTTP 204 (No Content).


## `/v1/webspace/config`
### GET
Get current webspace configuration values.

<a name="config-get-res"></a>Response:

HTTP 200 body containing the following:
 - `startupDelay`: Decimal representing how many seconds to delay incoming connections to a webspace while starting the
 container
 - `httpPort`: Incoming HTTP requests (and SSL-terminated HTTPS connections) will be forwarded to this port
 - `httpsPort`: If set to a non-zero value, SSL termination will be disabled an incoming HTTPS requests will be
 forwarded to this port

### PATCH
Update webspace configuration values.

Request:

See the [GET response](#config-get-res) for allowed request body values.

Response:

HTTP 200 with the previous values of any passed options.


## `/v1/webspace/domains`
### GET
Get the currently configured domains for the webspace.

Response:

HTTP 200 with an array of strings representing each configured domain.


## `/v1/webspace/domains/<domain>`
### POST
Add `domain` to the list of domains for the webspace.

Response:

HTTP 201 (Created)

### DELETE
Remove `domain` from the list of domains for the webspace.

Response:

HTTP 204 (No Content)


## `/v1/webspace/ports`
### GET
Obtain the current list of port forwardings for the webspace.

Response:

HTTP 200 with a map of external ports to internal ones.


## `/v1/webspace/ports/<ePort>/<iPort>`
### POST
Create a port forward from external port `ePort` to internal port `iPort`.

Response:

HTTP 201 (Created)

### DELETE
Remove a port forwarding from external port `ePort` to internal port `iPort`.

Response:

HTTP 204 (No Content)

## `/v1/webspace/ports/<iPort>`
### POST
Create a port forwarding from a random external port to internal port `<iPort>`.

Response:

HTTP 201 with the following:
 - `ePort`: Randomly selected external port
