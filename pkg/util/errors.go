package util

import (
	"errors"
	"net/http"
)

var (
	// ErrNotFound indicates an invalid API endpoint was provided
	ErrNotFound = errors.New("API endpoint not found")
	// ErrMethodNotAllowed indicates the user tried to use an API with an invalid request method
	ErrMethodNotAllowed = errors.New("method not allowed on API endpoint")
	// ErrTokenRequired indicates the user didn't pass an IAM token in the request
	ErrTokenRequired = errors.New("token required for this endpoint")
	// ErrAdminRequired indicates that an admin user is required
	ErrAdminRequired = errors.New("only admin users can make use of this endpoint")

	// ErrGenericNotFound indicates that a resource was not found
	ErrGenericNotFound = errors.New("not found")
	// ErrExists indicates that a resource already exists
	ErrExists = errors.New("already exists")
	// ErrUsed indicates that the requested resource is already in use by a webspace
	ErrUsed = errors.New("used by a webspace")
	// ErrNotRunning indicates that a webspace is not running
	ErrNotRunning = errors.New("not running")
	// ErrRunning indicates that a webspace is already running
	ErrRunning = errors.New("already running")
	// ErrDomainUnverified indicates that the request domain could not be verified
	ErrDomainUnverified = errors.New("verification failed")
	// ErrDefaultDomain indicates an attempt to remove the default domain
	ErrDefaultDomain = errors.New("cannot remove the default domain")
	// ErrTooManyPorts indicates that too many port forwards are configured
	ErrTooManyPorts = errors.New("port forward limit reached")
	// ErrBadPort indicates that the provided port is invalid
	ErrBadPort = errors.New("invalid port")
	// ErrInterface indicates the default interface is missing
	ErrInterface = errors.New("default network interface not present")
	// ErrAddress indicates the interface didn't have an IPv4 address
	ErrAddress = errors.New("IPv4 address not found")
	// ErrBadValue indicates an invalid value for a config option
	ErrBadValue = errors.New("invalid value for configuration option")
	// ErrUIDMismatch indicates the user ID didn't match that of the User object
	ErrUIDMismatch = errors.New("user id doesn't match provided value")
	// ErrTraefikProvider indicates an invalid Traefik config provider name was given
	ErrTraefikProvider = errors.New("invalid Traefik provider")
	// ErrWebsocket indicates the endpoint supports websocket communication only
	ErrWebsocket = errors.New("this endpoint supports websocket communication only")
)

// ErrToStatus converts an error to a HTTP status code
func ErrToStatus(err error) int {
	switch {
	case errors.Is(err, ErrTokenRequired), errors.Is(err, ErrAdminRequired):
		return http.StatusUnauthorized
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrGenericNotFound), errors.Is(err, ErrNotRunning):
		return http.StatusNotFound
	case errors.Is(err, ErrExists), errors.Is(err, ErrRunning), errors.Is(err, ErrUsed):
		return http.StatusConflict
	case errors.Is(err, ErrDomainUnverified), errors.Is(err, ErrBadPort),
		errors.Is(err, ErrTooManyPorts), errors.Is(err, ErrDefaultDomain),
		errors.Is(err, ErrBadValue), errors.Is(err, ErrWebsocket):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
