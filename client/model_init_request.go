/*
 * Netsoc webspaced
 *
 * API for managing next-gen webspaces. 
 *
 * API version: 1.2.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package webspaced
// InitRequest struct for InitRequest
type InitRequest struct {
	// Image alias or fingerprint
	Image string `json:"image"`
	// Password for root user
	Password string `json:"password,omitempty"`
	// Whether or not to install an SSH server (and create a port forward for it). Requires the user to have an SSH key on their account. 
	Ssh bool `json:"ssh,omitempty"`
}
