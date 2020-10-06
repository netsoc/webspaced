/*
 * Netsoc webspaced
 *
 * API for managing next-gen webspaces. 
 *
 * API version: 1.0.1
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package webspaced
// Usage Website resource usage
type Usage struct {
	// CPU time (nanoseconds)
	Cpu int64 `json:"cpu"`
	Disks map[string]int64 `json:"disks"`
	// Memory usage in bytes
	Memory int64 `json:"memory"`
	// Number of processes
	Processes int64 `json:"processes"`
}