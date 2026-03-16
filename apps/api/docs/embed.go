// Package docs expone la spec OpenAPI embebida para Swagger UI y ReDoc.
package docs

import _ "embed"

// SwaggerJSON es el contenido de la spec OpenAPI 3 (citaspot-api).
//go:embed swagger.json
var SwaggerJSON []byte
