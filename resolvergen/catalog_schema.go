package resolvergen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/stoewer/go-strcase"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

// catalogSchemaSuffix is the filename suffix of every generated per-schema catalog SDL file
const catalogSchemaSuffix = "catalog.graphql"

// catalogSchema holds the naming and list arguments for one catalog schema
type catalogSchema struct {
	// Name is the schema type name
	Name string
	// LowerName is the schema type name in lower camel case
	LowerName string
	// ListField is the connection list query field name for the schema
	ListField string
	// Arguments is the list query argument set rendered as SDL
	Arguments string
}

// WriteCatalogSchema writes one catalog SDL file per catalog schema into the graphql schema directory
func (r *ResolverPlugin) WriteCatalogSchema(schemaDir string) error {
	if schemaDir == "" {
		return ErrGraphSchemaDirRequired
	}

	if len(r.catalogSchemas) == 0 {
		return nil
	}

	listFields, err := loadConnectionQueryFields(schemaDir)
	if err != nil {
		return err
	}

	names := lo.Keys(r.catalogSchemas)
	slices.Sort(names)

	for _, name := range names {
		field, ok := listFields[name+Connection]
		if !ok {
			return fmt.Errorf("%w: %s", ErrCatalogListQueryNotFound, name)
		}

		content, err := renderCatalogSchema(catalogSchema{
			Name:      name,
			LowerName: strcase.LowerCamelCase(name),
			ListField: field.Name,
			Arguments: renderArguments(field.Arguments),
		})
		if err != nil {
			return err
		}

		if err := writeFileIfChanged(filepath.Join(schemaDir, catalogSchemaFilename(name)), content); err != nil {
			return err
		}
	}

	return nil
}

// catalogSchemaFilename returns the SDL filename for one catalog schema, e.g. entitycatalog.graphql
func catalogSchemaFilename(name string) string {
	return strings.ToLower(name) + catalogSchemaSuffix
}

// loadConnectionQueryFields parses the graphql schema directory and returns the query fields keyed by connection type
func loadConnectionQueryFields(schemaDir string) (map[string]*ast.FieldDefinition, error) {
	schemaFiles, err := filepath.Glob(filepath.Join(schemaDir, "*.graphql"))
	if err != nil {
		return nil, fmt.Errorf("list schema files: %w", err)
	}

	fields := map[string]*ast.FieldDefinition{}

	for _, path := range schemaFiles {
		if strings.HasSuffix(filepath.Base(path), catalogSchemaSuffix) {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read schema file %s: %w", path, err)
		}

		doc, err := parser.ParseSchema(&ast.Source{Name: path, Input: string(data)})
		if err != nil {
			return nil, fmt.Errorf("parse schema file %s: %w", path, err)
		}

		for _, def := range append(doc.Definitions, doc.Extensions...) {
			if !strings.EqualFold(def.Name, string(ast.Query)) {
				continue
			}

			for _, field := range def.Fields {
				if isConnectionListQuery(field) {
					fields[field.Type.Name()] = field
				}
			}
		}
	}

	return fields, nil
}

// isConnectionListQuery reports whether the field is the entity's connection list query, which carries a where argument
func isConnectionListQuery(field *ast.FieldDefinition) bool {
	return strings.HasSuffix(field.Type.Name(), Connection) && field.Arguments.ForName("where") != nil
}

// renderArguments renders the argument definitions as an inline SDL argument list
func renderArguments(args ast.ArgumentDefinitionList) string {
	rendered := make([]string, 0, len(args))

	for _, arg := range args {
		rendered = append(rendered, arg.Name+": "+arg.Type.String())
	}

	return strings.Join(rendered, ", ")
}

// renderCatalogSchema renders the catalog SDL template for one schema
func renderCatalogSchema(input catalogSchema) ([]byte, error) {
	t, err := template.New("catalog_schema.gotpl").ParseFS(templates, "templates/catalog_schema.gotpl")
	if err != nil {
		return nil, err
	}

	var content bytes.Buffer

	if err := t.Execute(&content, input); err != nil {
		return nil, err
	}

	return append(bytes.TrimSpace(content.Bytes()), '\n'), nil
}
