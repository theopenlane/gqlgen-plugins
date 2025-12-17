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

//go:embed templates/**.gotpl templates/**/**.gotpl
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
	// EntPackage is the ent import for the generated types, defaults to `generated`
	EntPackage string
	// GraphQLImport is the import path for the graphql package
	GraphQLImport string
	// IncludeCustomUpdateFields is a flag to include custom fields
	IncludeCustomUpdateFields bool
	// ArchivableSchemas is a map of entity names that support archived status filtering
	ArchivableSchemas map[string]bool
}

// renderTemplate renders the template with the given name
func renderTemplate(templateName string, input *crudResolver, childTemplates []string) string {
	patterns := []string{"templates/" + templateName}

	for _, child := range childTemplates {
		patterns = append(patterns, "templates/"+child)
	}

	t, err := template.New(templateName).Funcs(template.FuncMap{
		"getEntityName":           getEntityName,
		"getInputObjectName":      getInputObjectName,
		"toLower":                 strings.ToLower,
		"toLowerCamel":            strcase.LowerCamelCase,
		"hasArgument":             hasArgument,
		"isListType":              isListType,
		"hasOwnerField":           hasOwnerField,
		"reserveImport":           gqltemplates.CurrentImports.Reserve,
		"modelPackage":            modelPackage,
		"isCommentUpdateOnObject": isCommentUpdateOnObject,
		"contains":                strings.Contains,
		"hasStatusField":          func(entityName string) bool { return input.ArchivableSchemas[entityName] },
		"getArchivedStatusValue":  getArchivedStatusEnum,
	}).ParseFS(templates, patterns...)
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

// getEntPackageFromImport returns the package name from the ent import path
func getEntPackageFromImport(importPath string) string {
	parts := strings.Split(importPath, "/")

	pkg := parts[len(parts)-1]

	if pkg == "" {
		return "generated"
	}

	return pkg
}

// renderCreate renders the create template
func (r *ResolverPlugin) renderCreate(field *codegen.Field) string {
	return renderTemplate("create.gotpl", &crudResolver{
		Field:                     field,
		ModelPackage:              r.modelPackage,
		EntImport:                 r.entGeneratedPackage,
		EntPackage:                getEntPackageFromImport(r.entGeneratedPackage),
		GraphQLImport:             r.graphqlImport,
		IncludeCustomUpdateFields: r.includeCustomFields,
		ArchivableSchemas:         r.archivableSchemas,
	}, []string{})
}

// renderUpdate renders the update template
func (r *ResolverPlugin) renderUpdate(field *codegen.Field) string {
	appendFields := getAppendFields(field)

	cr := &crudResolver{
		Field:                     field,
		AppendFields:              appendFields,
		ModelPackage:              r.modelPackage,
		EntImport:                 r.entGeneratedPackage,
		EntPackage:                getEntPackageFromImport(r.entGeneratedPackage),
		GraphQLImport:             r.graphqlImport,
		IncludeCustomUpdateFields: r.includeCustomFields,
		ArchivableSchemas:         r.archivableSchemas,
	}

	return renderTemplate("update.gotpl", cr, []string{"updatefields/*.gotpl"})
}

// renderDelete renders the delete template
func (r *ResolverPlugin) renderDelete(field *codegen.Field) string {
	return renderTemplate("delete.gotpl", &crudResolver{
		Field:                     field,
		ModelPackage:              r.modelPackage,
		EntImport:                 r.entGeneratedPackage,
		EntPackage:                getEntPackageFromImport(r.entGeneratedPackage),
		GraphQLImport:             r.graphqlImport,
		IncludeCustomUpdateFields: r.includeCustomFields,
		ArchivableSchemas:         r.archivableSchemas,
	}, []string{"deletefields/*.gotpl"})
}

// renderBulkUpload renders the bulk upload template
func (r *ResolverPlugin) renderBulkUpload(field *codegen.Field) string {
	return renderTemplate("upload.gotpl", &crudResolver{
		Field:             field,
		ModelPackage:      r.modelPackage,
		EntImport:         r.entGeneratedPackage,
		EntPackage:        getEntPackageFromImport(r.entGeneratedPackage),
		GraphQLImport:     r.graphqlImport,
		ArchivableSchemas: r.archivableSchemas,
	}, []string{})
}

// renderBulk renders the bulk template
func (r *ResolverPlugin) renderBulk(field *codegen.Field) string {
	return renderTemplate("bulk.gotpl", &crudResolver{
		Field:             field,
		ModelPackage:      r.modelPackage,
		EntImport:         r.entGeneratedPackage,
		EntPackage:        getEntPackageFromImport(r.entGeneratedPackage),
		GraphQLImport:     r.graphqlImport,
		ArchivableSchemas: r.archivableSchemas,
	}, []string{})
}

// renderQuery renders the query template
func (r *ResolverPlugin) renderQuery(field *codegen.Field) string {
	return renderTemplate("get.gotpl", &crudResolver{
		Field:             field,
		ModelPackage:      r.modelPackage,
		EntImport:         r.entGeneratedPackage,
		EntPackage:        getEntPackageFromImport(r.entGeneratedPackage),
		GraphQLImport:     r.graphqlImport,
		ArchivableSchemas: r.archivableSchemas,
	}, []string{})
}

// renderList renders the list template
func (r *ResolverPlugin) renderList(field *codegen.Field) string {
	return renderTemplate("list.gotpl", &crudResolver{
		Field:             field,
		EntImport:         r.entGeneratedPackage,
		EntPackage:        getEntPackageFromImport(r.entGeneratedPackage),
		GraphQLImport:     r.graphqlImport,
		ArchivableSchemas: r.archivableSchemas,
	}, []string{})
}

const (
	CreateOperation  = "Create"
	UpdateOperation  = "Update"
	AddOperation     = "Add"
	DeleteOperation  = "Delete"
	InputObject      = "Input"
	BulkOperation    = "Bulk"
	CSVOperation     = "CSV"
	BulkCSVOperation = "BulkCSV"
	UploadOperation  = "Upload"
	Connection       = "Connection"
	Payload          = "Payload"
)

// crudTypes is a list of CRUD operations that are included in the resolver name
var stripStrings = []string{CreateOperation, UpdateOperation, DeleteOperation, BulkOperation, CSVOperation, UploadOperation, Connection, Payload}

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
	return args.ForName(arg) != nil
}

func isListType(arg string, args gqlast.ArgumentDefinitionList) bool {
	a := args.ForName(arg)
	if a == nil {
		return false
	}

	return a.Type.Elem != nil
}

// hasOwnerField checks if the field has an owner field in the input arguments
func hasOwnerField(field *codegen.Field) bool {
	if crudType(field) == CreateOperation {
		return argsHasOwnerID(field.Args)
	}

	// check the input of the create, instead of the update since its immutable
	checkFieldName := strings.Replace(field.Name, "update", "create", 1)

	// remove the Bulk and BulkCSV from fields
	checkFieldName = strings.ReplaceAll(checkFieldName, BulkOperation, "")
	checkFieldName = strings.ReplaceAll(checkFieldName, CSVOperation, "")

	if field.Object.HasField(checkFieldName) {
		for _, obj := range field.Object.Fields {
			if obj.Name == checkFieldName {
				return argsHasOwnerID(obj.Args)
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

// isCommentUpdateOnObject checks if the field is of the format "Update<Something>Comment"
func isCommentUpdateOnObject(field string) bool {
	if strings.Contains(field, UpdateOperation) && strings.Contains(field, "Comment") {
		return true
	}

	return false
}

// getInputObjectName returns the input object name by stripping the CRUD operation from the resolver name
// for example UpdateTaskInput will return Task
func getInputObjectName(objectName string) string {
	// replace all operations
	objectName = strings.ReplaceAll(objectName, CreateOperation, "")
	objectName = strings.ReplaceAll(objectName, UpdateOperation, "")

	return strings.ReplaceAll(objectName, InputObject, "")
}

// getArchivedStatusEnum returns the archived status enum value for the entity
func getArchivedStatusEnum(entityName string) string {
	return "enums." + entityName + "StatusArchived"
}
