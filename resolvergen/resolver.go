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
	// graphqlImport is the import path for the graphql package
	graphqlImport string
	// csvGeneratedPackage is the import path for the csvgenerated package with CSV wrapper types
	csvGeneratedPackage string
	// includeCustomFields includes custom resolver fields for updates, templates
	// are stored in `templates/updatefields/*.gotpl`, `templates/deletefields/*.gotpl`
	// defaults to true
	includeCustomFields bool
	// forceRegenerateBulkResolvers when true will overwrite existing bulk resolver
	// implementations with freshly generated code from templates. Use this for one-time
	// migrations when bulk templates change, then disable to preserve custom logic.
	forceRegenerateBulkResolvers bool

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

// WithGraphQLImport sets the import path for the graphql package
func WithGraphQLImport(graphqlImport string) Options {
	return func(p *ResolverPlugin) {
		p.graphqlImport = graphqlImport
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

// WithCSVGeneratedPackage sets the import path for the csvgenerated package
func WithCSVGeneratedPackage(pkg string) Options {
	return func(p *ResolverPlugin) {
		p.csvGeneratedPackage = pkg
	}
}

// WithForceRegenerateBulkResolvers enables forced regeneration of bulk resolver
// implementations, overwriting any existing custom logic. Use this for one-time
// migrations when bulk templates change, then disable to preserve customizations.
func WithForceRegenerateBulkResolvers(enabled bool) Options {
	return func(p *ResolverPlugin) {
		p.forceRegenerateBulkResolvers = enabled
	}
}

// Implement gqlgen api.ResolverImplementer
func (r *ResolverPlugin) Implement(s string, f *codegen.Field) (val string) {
	// if the field has a custom resolver, use it
	// panic is not a custom resolver so attempt to implement the field
	// regenerate bulk operations if forceRegenerateBulkResolvers is enabled
	if s != "" && !strings.Contains(s, "panic") && !r.shouldRegenerateBulkResolver(f) {
		return s
	}

	switch {
	case isMutation(f), isInput(f):
		return r.mutationImplementer(f)
	case isQuery(f):
		return r.queryImplementer(f)
	case isWorkflowResolverField(f):
		return r.workflowImplementer(f)
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
	if err := r.Plugin.GenerateCode(data); err != nil {
		return err
	}

	resolverDir := data.Config.Resolver.Dir()
	if resolverDir == "" {
		return nil
	}

	return UpdateWorkflowResolvers(resolverDir)
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

func isWorkflowResolverField(f *codegen.Field) bool {
	if f == nil || f.Object == nil {
		return false
	}

	if isMutation(f) || isQuery(f) || isInput(f) {
		return false
	}

	_, ok := workflowResolverHelpers[f.GoFieldName]

	return ok
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

func (r *ResolverPlugin) workflowImplementer(f *codegen.Field) string {
	helperName, ok := workflowResolverHelpers[f.GoFieldName]
	if !ok {
		return fmt.Sprintf(defaultImplementation, f.GoFieldName, f.Name)
	}

	objectType := ""
	if f.Object != nil {
		objectType = f.Object.Name
	}

	if objectType == "" {
		return fmt.Sprintf(defaultImplementation, f.GoFieldName, f.Name)
	}

	return renderWorkflowTemplate(&workflowResolverTemplate{
		HelperName: helperName,
		ObjectType: objectType,
		EntPackage: getEntPackageFromImport(r.entGeneratedPackage),
		IsTimeline: helperName == "workflowResolverTimeline",
	})
}

// shouldRegenerateBulkResolver returns true if the resolver should be forcefully regenerated.
// This is used to update existing bulk resolvers to use the latest templates with CSV wrapper types.
// The behavior is controlled by the forceRegenerateBulkResolvers option.
func (r *ResolverPlugin) shouldRegenerateBulkResolver(f *codegen.Field) bool {
	if f == nil || !r.forceRegenerateBulkResolvers {
		return false
	}

	// Only regenerate actual bulk mutation resolvers, not extended resolvers or other fields
	// that happen to contain "Bulk" or "CSV" in their names
	fieldName := f.GoFieldName

	// Check for specific bulk mutation patterns (createBulk*, updateBulk*, deleteBulk*)
	isBulkMutation := strings.HasPrefix(fieldName, "CreateBulk") ||
		strings.HasPrefix(fieldName, "UpdateBulk") ||
		strings.HasPrefix(fieldName, "DeleteBulk")

	return isBulkMutation
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
