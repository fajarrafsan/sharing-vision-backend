// Package api menyimpan spesifikasi OpenAPI yang ikut di-embed ke binary.
package api

import _ "embed"

//go:embed openapi.yaml
var OpenAPI []byte
