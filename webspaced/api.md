# REST API
`webspaced` exposes a REST API for managing webspace containers.

# Authentication
Since the API is exposed over a Unix socket, it is possible to authenticate a client by obtaining their UID via
`SO_PEERCRED`. If the user is root (UID 0) or a member of the `webspace-admin` group, the desired user may specified
in the `X-Webspace-User` header.

If the API is to be exposed over TCP in future, an additional authentication mechanism would be added.

# Errors
In an error scenario, an endpoint will return an appropriate 4XX (client error) or 5XX (server error) HTTP status, along
with a JSON object containing a `message` string for display to a user. Additional fields _may_ be provided.

# Endpoints
## `/v1/images`
### GET
Obtain a list of available images.

Response:

HTTP 200 and a list of objects containing the following properties from the response
[here](https://github.com/lxc/lxd/blob/master/doc/rest-api.md#get-optional-secretsecret):
 - `aliases`
 - `fingerprint`
 - `properties`
 - `size`

## `/v1/webspace`
### GET
Retrieve all information about a webspace (including [configuration](#config-get-res), [domains](#domains-get-res) and
[ports](#ports-get-res)).

Response:

HTTP 200 body containing the following:

- `user`: Name of the webspace user
- [`config`](#config-get-res)
- [`domains`](#domains-get-res)
- [`ports`](#ports-get-res)

Errors:
 - Webspace does not exist (HTTP 404 Not Found)

### POST
Initialize the user's webspace.

Request:

 - `image`: String representing LXD image by alias or fingerprint
 - `password`: _(Optional)_ Root password
 - `sshKey`: _(Optional)_ SSH public key to add for `root` - specifying this will ensure SSH is installed / enabled and
    add a forward for port 22

Response:

HTTP 201 Created and a response body identical to `GET /v1/webspace`.

Errors:
 - Webspace already exists (HTTP 409 Conflict)
 - Image not found (HTTP 404 Not Found)

### DELETE
Destroy the user's webspace.

Response:

HTTP 204 (No Content).

Errors:
 - Webspace does not exist (HTTP 404 Not Found)


## `/v1/webspace/state`
### POST
Start the webspace container.

Response:

HTTP 204 (No Content).

Errors:
 - Webspace does not exist (HTTP 404 Not Found)
 - Webspace already running (HTTP 409 Conflict)

### PUT
Reboot the webspace container.

Response:

HTTP 204 (No Content).

Errors:
 - Webspace does not exist (HTTP 404 Not Found)
 - Webspace not running (HTTP 404 Not Found)

### DELETE
Shutdown the webspace container.

Response:

HTTP 204 (No Content).

Errors:
 - Webspace does not exist (HTTP 404 Not Found)
 - Webspace not running (HTTP 404 Not Found)


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

Errors:
 - Webspace does not exist (HTTP 404 Not Found)

### PATCH
Update webspace configuration values.

Request:

See the [GET response](#config-get-res) for allowed request body values.

Response:

HTTP 200 with the previous values of any passed options.

Errors:
 - Webspace does not exist (HTTP 404 Not Found)
 - Invalid fields or incorrectly formatted fields (HTTP 400 Bad Request)


## `/v1/webspace/domains`
### GET
Get the currently configured domains for the webspace.

<a name="domains-get-res"></a>Response:

HTTP 200 with an array of strings representing each configured domain.

Errors:
 - Webspace does not exist (HTTP 404 Not Found)


## `/v1/webspace/domains/<domain>`
### POST
Add `domain` to the list of domains for the webspace.

Response:

HTTP 201 (Created)

Errors:
 - Webspace does not exist (HTTP 404 Not Found)
 - Domain already exists (HTTP 409 Conflict)
 - Domain verification failed (e.g. TXT verification record not found or DNS lookup failed) (HTTP 400 Bad Request / HTTP
 500 Internal Server Error)

### DELETE
Remove `domain` from the list of domains for the webspace.

Response:

HTTP 204 (No Content)

Errors:
 - Webspace does not exist (HTTP 404 Not Found)
 - Domain does not exist (HTTP 404 Not Found)
 - Attempt to delete default domain (HTTP 400 Bad Request)

## `/v1/webspace/ports`
### GET
Obtain the current list of port forwardings for the webspace.

<a name="ports-get-res"></a>Response:

HTTP 200 with a map of external ports to internal ones.

Errors:
 - Webspace does not exist (HTTP 404 Not Found)

## `/v1/webspace/ports/<ePort>/<iPort>`
### POST
Create a port forward from external port `ePort` to internal port `iPort`.

Response:

HTTP 201 (Created)

Errors:
 - Webspace does not exist (HTTP 404 Not Found)
 - Invalid port(s) (HTTP 400 Bad Request)
 - External port in use (HTTP 400 Bad Request)

### DELETE
Remove a port forwarding from external port `ePort` to internal port `iPort`.

Response:

HTTP 204 (No Content)

Errors:
 - Webspace does not exist (HTTP 404 Not Found)
 - Invalid port(s) (HTTP 400 Bad Request)
 - Port mapping does not exist (HTTP 404 Not Found)

## `/v1/webspace/ports/<iPort>`
### POST
Create a port forwarding from a random external port to internal port `<iPort>`.

Response:

HTTP 201 with the following:
 - `ePort`: Randomly selected external port

Errors:
 - Webspace does not exist (HTTP 404 Not Found)
 - Invalid port (HTTP 400 Bad Request)
