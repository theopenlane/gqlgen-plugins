package bulkgen

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/plugin"
	"github.com/gertd/go-pluralize"
	"github.com/rs/zerolog/log"
	"github.com/stoewer/go-strcase"
)

//go:embed bulk.gotpl
var bulkTemplate string

// New returns a new plugin
func New() plugin.Plugin {
	return &Plugin{}
}

// Options is a function to set the options for the plugin
type Options func(*Plugin)

// NewWithOptions returns a new plugin with the given options
func NewWithOptions(opts ...Options) *Plugin {
	r := &Plugin{}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// WithModelPackage sets the model package for the gqlgen model
func WithModelPackage(modelPackage string) Options {
	return func(p *Plugin) {
		p.ModelPackage = modelPackage
	}
}

// WithEntGeneratedPackage sets the ent generated package for the gqlgen model
func WithEntGeneratedPackage(entPackage string) Options {
	return func(p *Plugin) {
		p.EntGeneratedPackage = entPackage
	}
}

// WithCSVOutputPath sets the file path location that CSVs are written to
func WithCSVOutputPath(path string) Options {
	return func(p *Plugin) {
		p.CSVOutputPath = path
	}
}

// Plugin is a gqlgen plugin to generate bulk resolver functions used for mutations
type Plugin struct {
	// ModelPackage is the package name for the gqlgen model
	ModelPackage string
	// EntGeneratedPackage is the ent generated package that holds the generated types
	EntGeneratedPackage string
	// CSVOutputPath is the file path location that CSVs are written to
	CSVOutputPath string
}

// Name returns the name of the plugin
func (m *Plugin) Name() string {
	return "bulkgen"
}

// BulkResolverBuild is a struct to hold the objects for the bulk resolver
type BulkResolverBuild struct {
	// Objects is a list of objects to generate bulk resolvers for
	Objects []Object
	// ModelImport is the import path for the gqlgen model
	ModelImport string
	// EntImport is the ent generated package that holds the generated types
	EntImport string
	// ModelPackage is the package name for the gqlgen model
	ModelPackage string
}

// Object is a struct to hold the object name for the bulk resolver
type Object struct {
	// Name of the object
	Name string
	// PluralName of the object
	PluralName string
	// Fields of the object
	Fields []string
	// OperationType indicates whether this is a create or delete operation
	OperationType string
}

// GenerateCode generates the bulk resolver code
func (m *Plugin) GenerateCode(data *codegen.Data) error {
	if !data.Config.Resolver.IsDefined() {
		return nil
	}

	return m.generateSingleFile(*data)
}

// generateSingleFile generates the bulk resolver code, this is all done in a single file and
// used by the resolvergen plugin for each bulk resolver
func (m *Plugin) generateSingleFile(data codegen.Data) error {
	inputData := BulkResolverBuild{
		Objects:     []Object{},
		ModelImport: m.ModelPackage,
		EntImport:   m.EntGeneratedPackage,
	}

	// only add the model package if the import is not empty
	if m.ModelPackage != "" {
		modelPkg := data.Config.Model.Package
		if modelPkg != "" {
			modelPkg += "."
		}

		inputData.ModelPackage = modelPkg
	}

	if m.CSVOutputPath == "" {
		m.CSVOutputPath = data.Config.Resolver.Dir() + "/csv"
	}

	// create the directory if it does not exist
	if _, err := os.Stat(m.CSVOutputPath); os.IsNotExist(err) {
		if err := os.MkdirAll(m.CSVOutputPath, os.ModePerm); err != nil {
			return err
		}
	}

	for _, f := range data.Schema.Mutation.Fields {
		lowerName := strings.ToLower(f.Name)

		// if the field is a bulk create or delete mutation, add it to the list of objects
		// we skip csv bulk mutations because they will reuse the same functions
		if strings.Contains(lowerName, "bulk") && !strings.Contains(lowerName, "csv") {
			var objectName, operationType string

			switch {
			case strings.Contains(lowerName, "createbulk"):
				objectName = strings.Replace(f.Name, "createBulk", "", 1)
				operationType = "create"
			case strings.Contains(lowerName, "bulkcreate"):
				objectName = strings.Replace(f.Name, "bulkCreate", "", 1)
				operationType = "create"
			case strings.Contains(lowerName, "deletebulk"):
				objectName = strings.Replace(f.Name, "deleteBulk", "", 1)
				operationType = "delete"
			case strings.Contains(lowerName, "bulkdelete"):
				objectName = strings.Replace(f.Name, "bulkDelete", "", 1)
				operationType = "delete"
			default:
				continue
			}

			object := Object{
				Name:          objectName,
				PluralName:    pluralize.NewClient().Plural(objectName),
				Fields:        getCreateInputFields(objectName, data),
				OperationType: operationType,
			}

			inputData.Objects = append(inputData.Objects, object)

			// Generate and write the CSV file only for create operations
			if operationType == "create" {
				if err := generateSampleCSV(object, m.CSVOutputPath); err != nil {
					return err
				}
			}
		}
	}

	// render the bulk resolver template
	return templates.Render(templates.Options{
		PackageName: data.Config.Resolver.Package,            // use the resolver package
		Filename:    data.Config.Resolver.Dir() + "/bulk.go", // write to the resolver directory
		FileNotice:  `// THIS CODE IS REGENERATED BY github.com/theopenlane/core/pkg/gqlplugin. DO NOT EDIT.`,
		Data:        inputData,
		Funcs: template.FuncMap{
			"toLower": strings.ToLower,
		},
		Packages: data.Config.Packages,
		Template: bulkTemplate,
	})
}

// getCreateInputFields returns the list of fields available in the Create<object>Input
func getCreateInputFields(objectName string, data codegen.Data) (inputFields []string) {
	inputTypeName := "Create" + objectName + "Input"
	if inputType, ok := data.Schema.Types[inputTypeName]; ok {
		for _, f := range inputType.Fields {
			inputFields = append(inputFields, strcase.UpperCamelCase(f.Name))
		}
	}

	return inputFields
}

// generateSampleCSV generates a sample CSV file for the given object
func generateSampleCSV(object Object, outputPath string) error {
	headers := object.Fields

	filePath := fmt.Sprintf("%s/sample_%s.csv", outputPath, strings.ToLower(object.Name))

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}

	defer file.Close()

	writer := csv.NewWriter(file)

	defer writer.Flush()

	if err := writer.Write(headers); err != nil {
		return err
	}

	exampleRow := make([]string, len(headers))
	for i := range headers {
		exampleRow[i] = fmt.Sprintf("example_%s", strings.ToLower(headers[i]))
	}

	if err := writer.Write(exampleRow); err != nil {
		return err
	}

	log.Debug().Msgf("Sample CSV for %s created: %s", object.Name, filePath)

	return nil
}
