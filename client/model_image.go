/*
 * Netsoc webspaced
 *
 * API for managing next-gen webspaces. 
 *
 * API version: 1.0.1
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package webspaced
// Image LXD image (summarised version of https://linuxcontainers.org/lxd/docs/master/rest-api#10imagesfingerprint) 
type Image struct {
	Aliases []ImageAlias `json:"aliases"`
	// SHA-256 hash of the image
	Fingerprint string `json:"fingerprint"`
	// Arbitrary properties
	Properties map[string]string `json:"properties"`
	// Size in bytes
	Size int64 `json:"size"`
}
