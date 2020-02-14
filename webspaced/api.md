# REST API
`webspaced` exposes a REST API for managing webspace containers.

# Authentication
Since the API is exposed over a Unix socket, it is possible to authenticate a client by obtaining their UID via
`SO_PEERCRED`. If the user is root (UID 0) or a member of the `webspace-admin` group, the desired user may specified
in the `X-Webspace-User` header.

If the API is to be exposed over TCP in future, an additional authentication mechanism would be added.

# Endpoints
## `/v1/config`
### POST
Initialize the user's webspace.

Request:
 - `image`: String representing LXD image by alias or fingerprint
 - `password`: _(Optional)_ Root password
 - `sshKey`: _(Optional)_ SSH public key to add for `root` - specifying this will ensure SSH is installed / enabled and
    add a forward for port 22

Response:

Either HTTP 204 (No Content) and an empty response body (if `sshKey` was not present in the request) or HTTP 200 and
the following:
 - `sshPort`: The external port for SSH access
