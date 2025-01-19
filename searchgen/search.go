package searchgen

import (
	"cmp"
	_ "embed"
	"fmt"
	"slices"
	"strings"
	"text/template"

	"entgo.io/contrib/entgql"
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/gertd/go-pluralize"
	"github.com/theopenlane/entx/genhooks"
)

const (
	relativeSchemaPath = "./internal/ent/schema"
)

var defaultIDFields = []string{"ID", "DisplayID"}

var SearchDirective = entgql.NewDirective("search")

//go:embed templates/helpers.gotpl
var helperTemplate string

//go:embed templates/resolver.gotpl
var resolverTemplate string

// SearchPlugin is a gqlgen plugin to generate search functions
type SearchPlugin struct {
	EntGeneratedPackage string
	ModelPackage        string
	// IDFields are the fields that are searchable by ID, these are search by equals and not by like
	// defaults to ID, DisplayID
	IDFields []string
}

// Name returns the name of the plugin
func (r SearchPlugin) Name() string {
	return "searchgen"
}

// NewSearchPlugin returns a new search plugin
func New(entPackage string) *SearchPlugin {
	return &SearchPlugin{
		EntGeneratedPackage: entPackage,
	}
}

// Options is a function to set the options for the plugin
type Options func(*SearchPlugin)

// NewWithOptions returns a new search plugin with the given options
func NewWithOptions(opts ...Options) *SearchPlugin {
	r := &SearchPlugin{}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// WithModelPackage sets the model package for the gqlgen model
func WithModelPackage(modelPackage string) Options {
	return func(p *SearchPlugin) {
		p.ModelPackage = modelPackage
	}
}

// WithEntGeneratedPackage sets the ent generated package for the gqlgen model
func WithEntGeneratedPackage(entPackage string) Options {
	return func(p *SearchPlugin) {
		p.EntGeneratedPackage = entPackage
	}
}

func WithIDFields(fields []string) Options {
	return func(p *SearchPlugin) {
		p.IDFields = fields
	}
}

// SearchResolverBuild is a struct to hold the objects for the bulk resolver
type SearchResolverBuild struct {
	// Name of the search type
	Name string
	// Objects is a list of objects to generate bulk resolvers for
	Objects []Object
	// EntImport is the ent generated package that holds the generated types
	EntImport string
	// ModelImport is the package name for the gqlgen model
	ModelImport string
	// ModelPackage is the package name for the gqlgen model
	ModelPackage string
	// IDFields are the fields that are searchable by ID, these are search by equals and not by like
	// defaults to ID, DisplayID
	IDFields []string
}

// Object is a struct to hold the object name for the bulk resolver
type Object struct {
	// Name of the object
	Name string
	// Fields of the object that are searchable
	Fields []genhooks.Field
	// AdminFields of the object that are searchable
	AdminFields []genhooks.Field
}

// GenerateCode implements api.CodeGenerator to generate the search resolver and it's helper functions
func (r SearchPlugin) GenerateCode(data *codegen.Data) error {
	inputData, err := getInputData(data)
	if err != nil {
		return err
	}

	inputData.ModelImport = r.ModelPackage

	// only add the model package if the import is not empty
	if r.ModelPackage != "" {
		modelPkg := data.Config.Model.Package
		if modelPkg != "" {
			modelPkg += "."
		}

		inputData.ModelPackage = modelPkg
	}

	// add the generated package name
	inputData.EntImport = r.EntGeneratedPackage

	// set the default ID fields
	inputData.IDFields = defaultIDFields
	if r.IDFields != nil {
		inputData.IDFields = r.IDFields
	}

	// generate the search helper
	if err := genSearchHelper(data, inputData); err != nil {
		return err
	}

	// generate the search resolver
	inputData.Name = "Global"
	if err := genSearchResolver(data, inputData, "search"); err != nil {
		return err
	}

	// generate the admin search resolver
	inputData.Name = "Admin"

	return genSearchResolver(data, inputData, "adminsearch")
}

func getInputData(data *codegen.Data) (SearchResolverBuild, error) {
	inputData := SearchResolverBuild{
		Objects: []Object{},
	}

	graph, err := entc.LoadGraph(relativeSchemaPath, &gen.Config{})
	if err != nil {
		return inputData, err
	}

	for _, f := range data.Schema.Types {
		// Add the search fields
		if strings.Contains(f.Name, "Search") && !strings.Contains(f.Name, "GlobalSearch") {
			schemaName := strings.TrimSuffix(f.Name, "SearchResult")
			fields, adminFields := genhooks.GetSearchableFields(schemaName, graph)

			if len(fields) > 0 {
				inputData.Objects = append(inputData.Objects, Object{
					Name:        schemaName,
					Fields:      fields,      // add the fields that are being searched
					AdminFields: adminFields, // add the admin fields that are being searched
				})
			}
		}
	}

	// sort objects by name so we have consistent output
	slices.SortFunc(inputData.Objects, func(a, b Object) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return inputData, nil
}

// genSearchHelper generates the search helper functions
func genSearchHelper(data *codegen.Data, inputData SearchResolverBuild) error {
	return templates.Render(templates.Options{
		PackageName: data.Config.Resolver.Package,              // use the resolver package
		Filename:    data.Config.Resolver.Dir() + "/search.go", // write to the resolver directory
		FileNotice:  `// THIS CODE IS REGENERATED BY github.com/theopenlane/gqlgen-plugins. DO NOT EDIT.`,
		Data:        inputData,
		Funcs: template.FuncMap{
			"toLower":   strings.ToLower,
			"toPlural":  pluralize.NewClient().Plural,
			"isIDField": isIDField,
		},
		Packages: data.Config.Packages,
		Template: helperTemplate,
	})
}

// genSearchResolver generates the search resolver functions
func genSearchResolver(data *codegen.Data, inputData SearchResolverBuild, resolverName string) error {
	return templates.Render(templates.Options{
		PackageName: data.Config.Resolver.Package,                                               // use the resolver package
		Filename:    data.Config.Resolver.Dir() + fmt.Sprintf("/%s.resolvers.go", resolverName), // write to the resolver directory
		FileNotice:  `// THIS CODE IS REGENERATED BY github.com/theopenlane/gqlgen-plugins. DO NOT EDIT.`,
		Data:        inputData,
		Funcs: template.FuncMap{
			"toLower":  strings.ToLower,
			"toPlural": pluralize.NewClient().Plural,
		},
		Packages: data.Config.Packages,
		Template: resolverTemplate,
	})
}

// isIDField checks if the field is an ID field
func isIDField(f string, idFields []string) bool {
	for _, idField := range idFields {
		if f == idField {
			return true
		}
	}

	return false
}
