package resolvergen

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
	"github.com/theopenlane/entx"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/theopenlane/gqlgen-plugins/groupby"
)

const (
	// groupBySchemaFilename is the filename of the generated group by SDL file
	groupBySchemaFilename = "groupby.graphql"
	// GroupByOperation is the suffix of every generated group by query
	GroupByOperation = "GroupBy"
	// groupByConnectionSuffix is the suffix of the per schema group by query return type
	groupByConnectionSuffix = "GroupByConnection"
	// groupByBucketSuffix is the suffix of the per schema group by bucket type
	groupByBucketSuffix = "GroupByBucket"
	// whereArg is the list query filter argument
	whereArg = "where"
	// groupByFieldArg is the argument holding the field to group on
	groupByFieldArg = "field"
	// groupByFieldEnumSuffix is the suffix of the per schema group by field enum
	groupByFieldEnumSuffix = "GroupByField"
)

// groupByEntity holds the groupable fields of one schema
type groupByEntity struct {
	// Name is the schema type name
	Name string
	// Fields are the groupable field names
	Fields []string
}

// groupBySchemaEntity holds the naming used to render one schema's group by query
type groupBySchemaEntity struct {
	// Name is the schema type name
	Name string
	// QueryName is the group by query field name
	QueryName string
	// ListField is the connection list query field name for the schema
	ListField string
	// WhereInput is the where input type of the list query
	WhereInput string
	// ListArguments are the list query arguments without where, rendered as SDL for the bucket's nested connection
	ListArguments string
	// ListType is the list query return type, e.g. VulnerabilityConnection!
	ListType string
	// EnumValues are the group by field enum values
	EnumValues []string
}

// WriteGroupBySchema writes the group by SDL for every schema with groupable fields into the graphql schema directory
func (r *ResolverPlugin) WriteGroupBySchema(entSchemaPath, schemaDir string) error {
	if schemaDir == "" {
		return ErrGraphSchemaDirRequired
	}

	graph, err := entc.LoadGraph(entSchemaPath, &gen.Config{})
	if err != nil {
		return err
	}

	entities, err := groupByEntities(graph.Nodes)
	if err != nil {
		return err
	}

	return writeGroupBySchema(schemaDir, entities)
}

// groupByEntities returns the schemas with groupable fields sorted by name
func groupByEntities(nodes []*gen.Type) ([]groupByEntity, error) {
	entities := []groupByEntity{}

	for _, node := range nodes {
		entity := groupByEntity{Name: node.Name}

		for _, f := range node.Fields {
			annotation, ok := entx.GetAnnotation[*groupby.Annotation](f)
			if !ok || !annotation.Groupable {
				continue
			}

			if f.Type == nil || (f.Type.Type != field.TypeString && f.Type.Type != field.TypeEnum) {
				return nil, fmt.Errorf("%w: %s.%s", ErrGroupByFieldType, node.Name, f.Name)
			}

			entity.Fields = append(entity.Fields, f.Name)
		}

		if len(entity.Fields) > 0 {
			entities = append(entities, entity)
		}
	}

	slices.SortFunc(entities, func(a, b groupByEntity) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return entities, nil
}

// writeGroupBySchema renders the group by SDL file, removing it when no schema has groupable fields
func writeGroupBySchema(schemaDir string, entities []groupByEntity) error {
	path := filepath.Join(schemaDir, groupBySchemaFilename)

	if len(entities) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}

		return nil
	}

	listFields, err := loadConnectionQueryFields(schemaDir)
	if err != nil {
		return err
	}

	schemaEntities := make([]groupBySchemaEntity, 0, len(entities))

	for _, entity := range entities {
		listField, ok := listFields[entity.Name+Connection]
		if !ok {
			return fmt.Errorf("%w: %s", ErrGroupByListQueryNotFound, entity.Name)
		}

		enumValues := make([]string, 0, len(entity.Fields))
		for _, f := range entity.Fields {
			enumValues = append(enumValues, strings.ToUpper(f))
		}

		listArgs := make(ast.ArgumentDefinitionList, 0, len(listField.Arguments))
		for _, arg := range listField.Arguments {
			if arg.Name != whereArg {
				listArgs = append(listArgs, arg)
			}
		}

		schemaEntities = append(schemaEntities, groupBySchemaEntity{
			Name:          entity.Name,
			QueryName:     listField.Name + GroupByOperation,
			ListField:     listField.Name,
			WhereInput:    listField.Arguments.ForName(whereArg).Type.Name(),
			ListArguments: renderArguments(listArgs),
			ListType:      listField.Type.String(),
			EnumValues:    enumValues,
		})
	}

	content, err := renderSchemaTemplate("groupby_schema.gotpl", schemaEntities)
	if err != nil {
		return err
	}

	return writeFileIfChanged(path, content)
}
