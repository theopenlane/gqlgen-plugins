package resolvergen

import (
	"fmt"
	"strings"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/plugin"
	"github.com/99designs/gqlgen/plugin/resolvergen"
	"github.com/vektah/gqlparser/v2/ast"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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
	// modelPackage is the model package that holds the generated models for gql
	modelPackage string
	// entGeneratedPackage is the ent generated package that holds the generated types
	entGeneratedPackage string
	// includeCustomFields includes custom resolver fields for updates, templates
	// are stored in `templates/updatefields/*.gotpl`, `templates/deletefields/*.gotpl`
	// defaults to true
	includeCustomFields bool

	archivableSchemas map[string]bool
}

// Name returns the name of the plugin
// This name must match the upstream resolvergen to replace during code generation
func (r ResolverPlugin) Name() string {
	return "resolvergen"
}

// New returns a new resolver plugin
func New() *ResolverPlugin {
	return &ResolverPlugin{
		includeCustomFields: true,
	}
}

// NewWithOptions returns a new plugin with the given options
func NewWithOptions(opts ...Options) *ResolverPlugin {
	r := New()

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Options is a function to set the options for the plugin
type Options func(*ResolverPlugin)

// WithEntGeneratedPackage sets the ent generated package for imports
func WithEntGeneratedPackage(entPackage string) Options {
	return func(p *ResolverPlugin) {
		p.entGeneratedPackage = entPackage
	}
}

// WithExcludeCustomUpdateFields excludes custom resolver fields for updates resolvers
func WithExcludeCustomUpdateFields() Options {
	return func(p *ResolverPlugin) {
		p.includeCustomFields = false
	}
}

// WithArchivableSchemas sets schemas that can have a status of archived
func WithArchivableSchemas(schemas []string) Options {
	return func(p *ResolverPlugin) {
		p.archivableSchemas = map[string]bool{}

		for _, s := range schemas {
			p.archivableSchemas[cases.Title(language.English, cases.Compact).String(s)] = true
		}
	}
}

// Implement gqlgen api.ResolverImplementer
func (r *ResolverPlugin) Implement(s string, f *codegen.Field) (val string) {
	// if the field has a custom resolver, use it
	// panic is not a custom resolver so attempt to implement the field
	if s != "" && !strings.Contains(s, "panic") {
		return s
	}

	switch {
	case isMutation(f), isInput(f):
		return r.mutationImplementer(f)
	case isQuery(f):
		return r.queryImplementer(f)
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
	return strings.EqualFold(f.Object.Name, string(ast.Mutation))
}

// isQuery returns true if the field is a query
func isQuery(f *codegen.Field) bool {
	return strings.EqualFold(f.Object.Name, string(ast.Query))
}

// isQuery returns true if the field is a query
func isInput(f *codegen.Field) bool {
	return strings.Contains(f.Object.Name, "Input")
}

// mutationImplementer returns the implementation for the mutation
func (r *ResolverPlugin) mutationImplementer(f *codegen.Field) string {
	switch crudType(f) {
	case BulkCSVOperation:
		return r.renderBulkUpload(f)
	case BulkOperation:
		return r.renderBulk(f)
	case UploadOperation:
		return r.renderBulkUpload(f)
	case CreateOperation:
		return r.renderCreate(f)
	case UpdateOperation:
		return r.renderUpdate(f)
	case DeleteOperation:
		return r.renderDelete(f)
	case InputObject:
		// this is needed to handle input fields that are not CRUD operations
		// first case is RevisionBump - might need to extend for others later
		return r.renderUpdate(f)
	default:
		return fmt.Sprintf(defaultImplementation, f.GoFieldName, f.Name)
	}
}

// queryImplementer returns the implementation for the query
func (r *ResolverPlugin) queryImplementer(f *codegen.Field) string {
	if strings.Contains(f.TypeReference.Definition.Name, Connection) {
		return r.renderList(f)
	}

	return r.renderQuery(f)
}

// crudType returns the type of CRUD operation
func crudType(f *codegen.Field) string {
	switch {
	case strings.Contains(f.GoFieldName, CSVOperation):
		return BulkCSVOperation
	case strings.Contains(f.GoFieldName, BulkOperation):
		return BulkOperation
	case strings.Contains(f.GoFieldName, UploadOperation):
		return UploadOperation
	case strings.Contains(f.GoFieldName, CreateOperation):
		return CreateOperation
	case strings.Contains(f.GoFieldName, UpdateOperation),
		// also include Add, which is an update to a parent object with a child object (e.g. add comment to a task)
		strings.Contains(f.GoFieldName, AddOperation):
		return UpdateOperation
	case strings.Contains(f.GoFieldName, DeleteOperation):
		return DeleteOperation
	// special case for input types that need mapped
	case f.TypeReference.Definition.Kind == ast.Scalar:
		return InputObject
	default:
		return "unknown"
	}
}
