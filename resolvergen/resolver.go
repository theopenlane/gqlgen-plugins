package resolvergen

import (
	"fmt"
	"strings"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/plugin"
	"github.com/99designs/gqlgen/plugin/resolvergen"
)

var (
	_ plugin.ResolverImplementer = (*ResolverPlugin)(nil)
)

const (
	defaultImplementation = "panic(fmt.Errorf(\"not implemented: %v - %v\"))"
)

// ResolverPlugin is a gqlgen plugin to generate resolver functions
type ResolverPlugin struct {
	*resolvergen.Plugin

	modelPackage string
}

// Name returns the name of the plugin
// This name must match the upstream resolvergen to replace during code generation
func (r ResolverPlugin) Name() string {
	return "resolvergen"
}

// New returns a new resolver plugin
func New() *ResolverPlugin {
	return &ResolverPlugin{}
}

// Implement gqlgen api.ResolverImplementer
func (r *ResolverPlugin) Implement(s string, f *codegen.Field) (val string) {
	// if the field has a custom resolver, use it
	// panic is not a custom resolver so attempt to implement the field
	if s != "" && !strings.Contains(s, "panic") {
		return s
	}

	switch {
	case isMutation(f):
		return mutationImplementer(f, r.modelPackage)
	case isQuery(f):
		return queryImplementer(f, r.modelPackage)
	default:
		return fmt.Sprintf(defaultImplementation, f.GoFieldName, f.Name)
	}
}

// GenerateCode implements api.CodeGenerator
func (r *ResolverPlugin) GenerateCode(data *codegen.Data) error {
	// set the model package if it is different from the resolver package
	if data.Config.Resolver.Package != data.Config.Model.Package {
		r.modelPackage = data.Config.Model.Package
	}

	// use the default resolver plugin to generate the code
	return r.Plugin.GenerateCode(data)
}

// isMutation returns true if the field is a mutation
func isMutation(f *codegen.Field) bool {
	return f.Object.Definition.Name == "Mutation"
}

// isQuery returns true if the field is a query
func isQuery(f *codegen.Field) bool {
	return f.Object.Definition.Name == "Query"
}

// mutationImplementer returns the implementation for the mutation
func mutationImplementer(f *codegen.Field, modelPackage string) string {
	switch crudType(f) {
	case "BulkCSV":
		return renderBulkUpload(f, modelPackage)
	case "Bulk":
		return renderBulk(f, modelPackage)
	case "Create":
		return renderCreate(f, modelPackage)
	case "Update":
		return renderUpdate(f, modelPackage)
	case "Delete":
		return renderDelete(f, modelPackage)
	default:
		return fmt.Sprintf(defaultImplementation, f.GoFieldName, f.Name)
	}
}

// queryImplementer returns the implementation for the query
func queryImplementer(f *codegen.Field, modelPackage string) string {
	if strings.Contains(f.TypeReference.Definition.Name, "Connection") {
		return renderList(f, modelPackage)
	}

	return renderQuery(f, modelPackage)
}

// crudType returns the type of CRUD operation
func crudType(f *codegen.Field) string {
	switch {
	case strings.Contains(f.GoFieldName, "CSV"):
		return "BulkCSV"
	case strings.Contains(f.GoFieldName, "Bulk"):
		return "Bulk"
	case strings.Contains(f.GoFieldName, "Create"):
		return "Create"
	case strings.Contains(f.GoFieldName, "Update"):
		return "Update"
	case strings.Contains(f.GoFieldName, "Delete"):
		return "Delete"
	default:
		return "unknown"
	}
}
