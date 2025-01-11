package fieldgen

import (
	"fmt"
	"strings"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/rs/zerolog/log"
	"github.com/vektah/gqlparser/v2/ast"
)

var inputString = `
extend type %s {
	%s: %s
}`

var srcName = "generated-by-fieldgen-plugin/%s-%s.graphql"

// createAdditionalSource creates a new source for the additional field
// so it can be added to the graphql schema and retrieved by resolvergen
func createAdditionalSource(schemaName, newField, fieldType string) *ast.Source {
	return &ast.Source{
		Name:    strings.ToLower(fmt.Sprintf(srcName, schemaName, newField)),
		Input:   fmt.Sprintf(inputString, schemaName, newField, fieldType),
		BuiltIn: false,
	}
}

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

var scalarString = `scalar %s`

func createAdditionalTypeSource(fieldType string) *ast.Source {
	return &ast.Source{
		Name:    strings.ToLower(fmt.Sprintf(srcName, fieldType)),
		Input:   fmt.Sprintf(scalarString, fieldType),
		BuiltIn: false,
	}
}

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
