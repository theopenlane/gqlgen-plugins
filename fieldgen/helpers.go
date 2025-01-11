package fieldgen

import (
	"fmt"
	"strings"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/rs/zerolog/log"
	"github.com/vektah/gqlparser/v2/ast"
)

// extendString is the string to add a field to the a type in the schema
var extendString = `
extend type %s {
	%s: %s
}`

// scalarString is the string to add a scalar to the schema
var scalarString = `scalar %s`

// srcName is the name of the source file to add the additional field to
var srcName = "generated-by-fieldgen-plugin/%s-%s.graphql"

// skippers is a list of strings to skip when adding fields to the schema
var skippers = []string{"History", "Connection", "Edge", "Payload", "AuditLog"}

// skipSchema skips the schema if it contains any of the skippers
// we only want the actual schema types not the connection types
func skipSchema(name string, t *ast.Definition) bool {
	if t.Kind != ast.Object {
		return true
	}

	for _, skip := range skippers {
		if strings.Contains(name, skip) {
			return true
		}
	}

	return false
}

// createAdditionalTypeSource creates a new source for the additional type
func createAdditionalTypeSource(fieldType string) *ast.Source {
	return &ast.Source{
		Name:    strings.ToLower(fmt.Sprintf(srcName, fieldType)),
		Input:   fmt.Sprintf(scalarString, fieldType),
		BuiltIn: false,
	}
}

// createAdditionalSource creates a new source for the additional field
// so it can be added to the graphql schema and retrieved by resolvergen
func createAdditionalSource(schemaName, newField, fieldType string) *ast.Source {
	return &ast.Source{
		Name:    strings.ToLower(fmt.Sprintf(srcName, schemaName, newField)),
		Input:   fmt.Sprintf(extendString, schemaName, newField, fieldType),
		BuiltIn: false,
	}
}

// addCustomType adds a custom type to the types and sources (graphql schema)
func addCustomType(customType string, cfg *config.Config) {
	// add the custom type to the imports
	exists := false

	for _, types := range cfg.Schema.Types {
		if types.Name == customType {
			exists = true
		}
	}

	if !exists {
		cfg.Schema.Types[customType] = &ast.Definition{
			Name: customType,
			Kind: ast.Scalar,
		}

		src := createAdditionalTypeSource(customType)
		cfg.Sources = append(cfg.Sources, src)

		log.Debug().Str("type", customType).Msgf("added custom type to schema")
	}
}
