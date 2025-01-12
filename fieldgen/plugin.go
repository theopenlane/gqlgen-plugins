package fieldgen

import (
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/vektah/gqlparser/v2/ast"
)

// ExtraFields is a struct to hold the additional fields to add to the schema
type ExtraFields struct {
	// FieldDefs is a list of additional fields to add to the schema
	FieldDefs []AdditionalField
}

// AdditionalField is a struct to with details about the additional field to add to the schema
type AdditionalField struct {
	// Name of the field to add
	Name string
	// Type of the field to add
	Type string
	// CustomType is an non-standard go type to use for the field, if set will override the Type field
	// If the Scalar has already been defined manually, add it to Type instead, this will
	// programmatically add the scalar to the schema
	CustomType string
	// NonNull indicates if the field is required
	NonNull bool
	// Description of the field
	Description string
	// AddToSchemaWithName is the name of the schema to add the field to, if empty will add to all schemas
	// unless AddToSchemaWithExistingField is set
	AddToSchemaWithNames []string
	// AddToSchemaWithExistingField will add to any schema with the existing field, if empty will add to all schemas
	// unless AddToSchemaWithName is set
	AddToSchemaWithExistingField string
}

// NewExtraFieldsGen returns a new ExtraFields plugin
func NewExtraFieldsGen(fields []AdditionalField) *ExtraFields {
	return &ExtraFields{
		FieldDefs: fields,
	}
}

// Name returns the plugin name
func (f *ExtraFields) Name() string {
	return "fieldgen"
}

// MutateConfig satisfies the plugin interface
func (f *ExtraFields) MutateConfig(cfg *config.Config) error {
	for i, t := range cfg.Schema.Types {
		for _, f := range f.FieldDefs {
			fieldType := f.Type

			if f.CustomType != "" {
				addCustomType(f.CustomType, cfg)

				fieldType = f.CustomType
			}

			newField := &ast.FieldDefinition{
				Name:        f.Name,
				Description: f.Description,
				Type: &ast.Type{
					NamedType: fieldType,
					NonNull:   f.NonNull,
				},
			}

			for _, schemaName := range f.AddToSchemaWithNames {
				if i == schemaName {
					cfg.Schema.Types[i].Fields = append(cfg.Schema.Types[i].Fields, newField)

					src := createAdditionalSource(t.Name, f.Name, fieldType)
					cfg.Sources = append(cfg.Sources, src)
				}
			}

			if f.AddToSchemaWithExistingField != "" {
				if skipSchema(i, t) {
					continue
				}

				if t.Fields.ForName(f.AddToSchemaWithExistingField) != nil {
					cfg.Schema.Types[i].Fields = append(cfg.Schema.Types[i].Fields, newField)

					src := createAdditionalSource(t.Name, f.Name, fieldType)
					cfg.Sources = append(cfg.Sources, src)
				}
			}
		}
	}

	return nil
}
