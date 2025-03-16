package resolvergen

import (
	"embed"

	"bytes"
	"html/template"
	"strings"

	"github.com/99designs/gqlgen/codegen"
	gqltemplates "github.com/99designs/gqlgen/codegen/templates"

	"github.com/stoewer/go-strcase"
	gqlast "github.com/vektah/gqlparser/v2/ast"
)

//go:embed templates/**.gotpl
var templates embed.FS

// crudResolver is a struct to hold the field for the CRUD resolver
type crudResolver struct {
	// Field is the field for the CRUD resolver
	Field *codegen.Field
	// AppendFields is the list of fields that can be appended in the update mutation
	AppendFields []string
	// ModelPackage is the package name for the gqlgen model
	ModelPackage string
	// EntImport is the ent import for the generated types
	EntImport string
}

// renderTemplate renders the template with the given name
func renderTemplate(templateName string, input *crudResolver) string {
	t, err := template.New(templateName).Funcs(template.FuncMap{
		"getEntityName": getEntityName,
		"toLower":       strings.ToLower,
		"toLowerCamel":  strcase.LowerCamelCase,
		"hasArgument":   hasArgument,
		"hasOwnerField": hasOwnerField,
		"reserveImport": gqltemplates.CurrentImports.Reserve,
		"modelPackage":  modelPackage,
	}).ParseFS(templates, "templates/"+templateName)
	if err != nil {
		panic(err)
	}

	var code bytes.Buffer

	if err = t.Execute(&code, input); err != nil {
		panic(err)
	}

	return strings.Trim(code.String(), "\t \n")
}

func modelPackage(modelPackage string) string {
	if modelPackage == "" {
		return ""
	}

	return modelPackage + "."
}

// renderCreate renders the create template
func (r *ResolverPlugin) renderCreate(field *codegen.Field) string {
	return renderTemplate("create.gotpl", &crudResolver{
		Field:        field,
		ModelPackage: r.modelPackage,
		EntImport:    r.entGeneratedPackage,
	})
}

// renderUpdate renders the update template
func (r *ResolverPlugin) renderUpdate(field *codegen.Field) string {
	appendFields := getAppendFields(field)

	cr := &crudResolver{
		Field:        field,
		AppendFields: appendFields,
		ModelPackage: r.modelPackage,
		EntImport:    r.entGeneratedPackage,
	}

	return renderTemplate("update.gotpl", cr)
}

// renderDelete renders the delete template
func (r *ResolverPlugin) renderDelete(field *codegen.Field) string {
	return renderTemplate("delete.gotpl", &crudResolver{
		Field:        field,
		ModelPackage: r.modelPackage,
		EntImport:    r.entGeneratedPackage,
	})
}

// renderBulkUpload renders the bulk upload template
func (r *ResolverPlugin) renderBulkUpload(field *codegen.Field) string {
	return renderTemplate("upload.gotpl", &crudResolver{
		Field:        field,
		ModelPackage: r.modelPackage,
		EntImport:    r.entGeneratedPackage,
	})
}

// renderBulk renders the bulk template
func (r *ResolverPlugin) renderBulk(field *codegen.Field) string {
	return renderTemplate("bulk.gotpl", &crudResolver{
		Field:        field,
		ModelPackage: r.modelPackage,
		EntImport:    r.entGeneratedPackage,
	})
}

// renderQuery renders the query template
func (r *ResolverPlugin) renderQuery(field *codegen.Field) string {
	return renderTemplate("get.gotpl", &crudResolver{
		Field:        field,
		ModelPackage: r.modelPackage,
		EntImport:    r.entGeneratedPackage,
	})
}

// renderList renders the list template
func (r *ResolverPlugin) renderList(field *codegen.Field) string {
	return renderTemplate("list.gotpl", &crudResolver{
		Field:     field,
		EntImport: r.entGeneratedPackage,
	})
}

const (
	CreateOperation  = "Create"
	UpdateOperation  = "Update"
	DeleteOperation  = "Delete"
	InputObject      = "Input"
	BulkOperation    = "Bulk"
	CSVOperation     = "CSV"
	BulkCSVOperation = "BulkCSV"
	Connection       = "Connection"
	Payload          = "Payload"
)

// crudTypes is a list of CRUD operations that are included in the resolver name
var stripStrings = []string{CreateOperation, UpdateOperation, DeleteOperation, BulkOperation, CSVOperation, Connection, Payload}

// getEntityName returns the entity name by stripping the CRUD operation from the resolver name
func getEntityName(name string) string {
	for _, s := range stripStrings {
		if strings.Contains(name, s) {
			name = strings.ReplaceAll(name, s, "")
		}
	}

	return name
}

// hasArgument checks if the argument is present in the list of arguments
func hasArgument(arg string, args gqlast.ArgumentDefinitionList) bool {
	for _, a := range args {
		if a.Name == arg {
			return true
		}
	}

	return false
}

// hasOwnerField checks if the field has an owner field in the input arguments
func hasOwnerField(field *codegen.Field) bool {
	if crudType(field) == CreateOperation {
		return argsHasOwnerID(field.Args)
	} else {
		// check the input of the create, instead of the update since its immutable
		checkFieldName := strings.Replace(field.Name, "update", "create", 1)

		if field.Object.HasField(checkFieldName) {
			for _, obj := range field.Object.Fields {
				if obj.Name == checkFieldName {
					return argsHasOwnerID(obj.Args)
				}
			}
		}
	}

	return false
}

const ownerIDField = "ownerID"

func argsHasOwnerID(args []*codegen.FieldArgument) bool {
	for _, arg := range args {
		if arg.TypeReference.Definition.Kind == gqlast.InputObject {
			if arg.TypeReference.Definition.Fields.ForName(ownerIDField) != nil {
				return true
			}
		}
	}

	return false
}

// getAppendFields returns the list of fields that are appendable in the update mutation
func getAppendFields(field *codegen.Field) (appendFields []string) {
	for _, arg := range field.Args {
		if arg.TypeReference.Definition.Kind == gqlast.InputObject {
			for _, f := range arg.TypeReference.Definition.Fields {
				if strings.Contains(f.Name, "append") {
					appendFields = append(appendFields, strcase.UpperCamelCase(f.Name))
				}
			}
		}
	}

	return
}
