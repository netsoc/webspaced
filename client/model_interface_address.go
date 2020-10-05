/*
 * Netsoc webspaced
 *
 * API for managing next-gen webspaces. 
 *
 * API version: 1.0.1
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package webspaced
// InterfaceAddress Network interface address
type InterfaceAddress struct {
	Family string `json:"family"`
	Address string `json:"address"`
	Netmask string `json:"netmask"`
	Scope string `json:"scope,omitempty"`
}
