package schema

import _ "embed"

//go:embed schema.sql
var schemaData []byte

func GetSchemaScript() string {
	return string(schemaData)
}
