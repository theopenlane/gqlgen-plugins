package resolvergen

import (
	"os"
	"path/filepath"
	"testing"

	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"

	"github.com/theopenlane/gqlgen-plugins/groupby"
)

const fakeGroupByListSchema = `type Query {
  vulnerabilities(after: Cursor, first: Int, before: Cursor, last: Int, orderBy: [VulnerabilityOrder!], where: VulnerabilityWhereInput): VulnerabilityConnection!
  findings(after: Cursor, first: Int, before: Cursor, last: Int, where: FindingWhereInput): FindingConnection!
}
`

func groupableField(name string, fieldType field.Type) *gen.Field {
	return &gen.Field{
		Name:        name,
		Type:        &field.TypeInfo{Type: fieldType},
		Annotations: gen.Annotations{groupby.AnnotationName: map[string]any{"groupable": true}},
	}
}

func groupByQueryField() *codegen.Field {
	return &codegen.Field{
		FieldDefinition: &ast.FieldDefinition{Name: "vulnerabilitiesGroupBy"},
		GoFieldName:     "VulnerabilitiesGroupBy",
		TypeReference:   &config.TypeReference{Definition: &ast.Definition{Name: "VulnerabilityGroupByConnection"}},
		Object:          &codegen.Object{Definition: &ast.Definition{Name: "Query"}},
		Args: []*codegen.FieldArgument{
			{
				ArgumentDefinition: &ast.ArgumentDefinition{Name: "field"},
				TypeReference: &config.TypeReference{Definition: &ast.Definition{
					Name:       "VulnerabilityGroupByField",
					Kind:       ast.Enum,
					EnumValues: ast.EnumValueList{{Name: "CVE_ID"}, {Name: "EXTERNAL_OWNER_ID"}},
				}},
			},
			{ArgumentDefinition: &ast.ArgumentDefinition{Name: "where"}},
		},
	}
}

func groupByRowsField() *codegen.Field {
	return &codegen.Field{
		FieldDefinition: &ast.FieldDefinition{
			Name: "vulnerabilities",
			Arguments: ast.ArgumentDefinitionList{
				{Name: "after", Type: ast.NamedType("Cursor", nil)},
				{Name: "first", Type: ast.NamedType("Int", nil)},
				{Name: "before", Type: ast.NamedType("Cursor", nil)},
				{Name: "last", Type: ast.NamedType("Int", nil)},
				{Name: "orderBy", Type: ast.ListType(ast.NonNullNamedType("VulnerabilityOrder", nil), nil)},
			},
		},
		GoFieldName:   "Vulnerabilities",
		TypeReference: &config.TypeReference{Definition: &ast.Definition{Name: "VulnerabilityConnection"}},
		Object:        &codegen.Object{Definition: &ast.Definition{Name: "VulnerabilityGroupByBucket"}},
	}
}

func TestGroupByEntities(t *testing.T) {
	nodes := []*gen.Type{
		{
			Name: "Vulnerability",
			Fields: []*gen.Field{
				groupableField("cve_id", field.TypeString),
				{Name: "display_name", Type: &field.TypeInfo{Type: field.TypeString}},
				groupableField("severity", field.TypeEnum),
			},
		},
		{
			Name:   "Control",
			Fields: []*gen.Field{{Name: "category", Type: &field.TypeInfo{Type: field.TypeString}}},
		},
		{
			Name:   "Finding",
			Fields: []*gen.Field{groupableField("external_owner_id", field.TypeString)},
		},
	}

	entities, err := groupByEntities(nodes)
	require.NoError(t, err)

	assert.Equal(t, []groupByEntity{
		{Name: "Finding", Fields: []string{"external_owner_id"}},
		{Name: "Vulnerability", Fields: []string{"cve_id", "severity"}},
	}, entities)
}

func TestGroupByEntitiesFieldType(t *testing.T) {
	nodes := []*gen.Type{
		{Name: "Vulnerability", Fields: []*gen.Field{groupableField("score", field.TypeFloat64)}},
	}

	_, err := groupByEntities(nodes)
	assert.ErrorIs(t, err, ErrGroupByFieldType)
}

func TestWriteGroupBySchema(t *testing.T) {
	t.Parallel()

	schemaDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(schemaDir, "ent.graphql"), []byte(fakeGroupByListSchema), 0o600)) // nolint:mnd

	entities := []groupByEntity{
		{Name: "Finding", Fields: []string{"external_owner_id"}},
		{Name: "Vulnerability", Fields: []string{"cve_id", "external_owner_id"}},
	}

	require.NoError(t, writeGroupBySchema(schemaDir, entities))

	content, err := os.ReadFile(filepath.Join(schemaDir, groupBySchemaFilename))
	require.NoError(t, err)

	doc, err := parser.ParseSchema(&ast.Source{Name: groupBySchemaFilename, Input: string(content)})
	require.NoError(t, err)

	enum := doc.Definitions.ForName("VulnerabilityGroupByField")
	require.NotNil(t, enum)
	assert.NotNil(t, enum.EnumValues.ForName("CVE_ID"))
	assert.NotNil(t, enum.EnumValues.ForName("EXTERNAL_OWNER_ID"))
	assert.NotNil(t, doc.Definitions.ForName("FindingGroupByField"))
	assert.NotNil(t, doc.Definitions.ForName("VulnerabilityGroupByConnection").Directives.ForName("goModel"))
	assert.NotNil(t, doc.Definitions.ForName("FindingGroupByConnection"))
	assert.NotNil(t, doc.Directives.ForName("goEnum"))

	orderField := doc.Definitions.ForName("GroupByOrderField")
	require.NotNil(t, orderField)
	assert.Equal(t, "github.com/theopenlane/gqlgen-plugins/groupby.OrderFieldKey", orderField.EnumValues.ForName("KEY").Directives.ForName("goEnum").Arguments.ForName("value").Value.Raw)

	bucket := doc.Definitions.ForName("VulnerabilityGroupByBucket")
	require.NotNil(t, bucket)

	rows := bucket.Fields.ForName("vulnerabilities")
	require.NotNil(t, rows)
	assert.Equal(t, "VulnerabilityConnection", rows.Type.Name())
	assert.True(t, rows.Type.NonNull)
	assert.NotNil(t, rows.Arguments.ForName("orderBy"))
	assert.NotNil(t, rows.Arguments.ForName("first"))
	assert.Nil(t, rows.Arguments.ForName("where"))
	assert.NotNil(t, rows.Directives.ForName("goField"))
	assert.NotNil(t, doc.Definitions.ForName("FindingGroupByBucket").Fields.ForName("findings"))

	query := doc.Extensions.ForName("Query")
	require.NotNil(t, query)

	vulnerabilities := query.Fields.ForName("vulnerabilitiesGroupBy")
	require.NotNil(t, vulnerabilities)
	assert.Equal(t, "VulnerabilityGroupByConnection", vulnerabilities.Type.Name())
	assert.Equal(t, "VulnerabilityWhereInput", vulnerabilities.Arguments.ForName("where").Type.Name())
	assert.Equal(t, "VulnerabilityGroupByField", vulnerabilities.Arguments.ForName("field").Type.Name())
	assert.NotNil(t, query.Fields.ForName("findingsGroupBy"))

	require.NoError(t, writeGroupBySchema(schemaDir, entities))

	require.NoError(t, writeGroupBySchema(schemaDir, nil))
	assert.NoFileExists(t, filepath.Join(schemaDir, groupBySchemaFilename))
}

func TestWriteGroupBySchemaErrors(t *testing.T) {
	t.Parallel()

	schemaDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(schemaDir, "ent.graphql"), []byte(fakeGroupByListSchema), 0o600)) // nolint:mnd

	err := writeGroupBySchema(schemaDir, []groupByEntity{{Name: "Program", Fields: []string{"name"}}})
	assert.ErrorIs(t, err, ErrGroupByListQueryNotFound)

	assert.ErrorIs(t, New().WriteGroupBySchema("./schema", ""), ErrGraphSchemaDirRequired)
}

func TestIsGroupByQuery(t *testing.T) {
	assert.True(t, isGroupByQuery(groupByQueryField()))
	assert.False(t, isGroupByQuery(catalogListField()))
	assert.False(t, isGroupByQuery(groupByRowsField()))
}

func TestIsGroupByRowsField(t *testing.T) {
	assert.True(t, isGroupByRowsField(groupByRowsField()))
	assert.False(t, isGroupByRowsField(groupByQueryField()))
	assert.False(t, isGroupByRowsField(catalogListField()))
}

func TestRenderListTemplateBucketFilter(t *testing.T) {
	field := groupByRowsField()

	scoped := renderTestTemplate(t, "list.gotpl", &crudResolver{
		Field:        field,
		EntImport:    "github.com/theopenlane/core/v2/internal/ent/generated",
		EntPackage:   "generated",
		Entity:       "Vulnerability",
		BucketFilter: true,
	})

	assert.Contains(t, scoped, "withTransactionalMutation(ctx).Vulnerability.Query().Where(predicate.Vulnerability(obj.Filter)).CollectFields(ctx)")
	assert.Contains(t, scoped, "generated.WithVulnerabilityOrder(orderBy)")
	assert.NotContains(t, scoped, "where.Filter")

	plain := renderTestTemplate(t, "list.gotpl", &crudResolver{
		Field:      field,
		EntImport:  "github.com/theopenlane/core/v2/internal/ent/generated",
		EntPackage: "generated",
		Entity:     "Vulnerability",
	})

	assert.NotContains(t, plain, "obj.Filter")
}

func TestGroupByResolverFields(t *testing.T) {
	entity, fields := groupByResolverFields(groupByQueryField())

	assert.Equal(t, "Vulnerability", entity)
	assert.Equal(t, []groupByResolverField{
		{EnumConstant: "VulnerabilityGroupByFieldCveID", Constant: "FieldCveID"},
		{EnumConstant: "VulnerabilityGroupByFieldExternalOwnerID", Constant: "FieldExternalOwnerID"},
	}, fields)
}

func TestRenderGroupByTemplate(t *testing.T) {
	entity, fields := groupByResolverFields(groupByQueryField())

	out := renderTestTemplate(t, "groupby.gotpl", &crudResolver{
		Field:         groupByQueryField(),
		EntImport:     "github.com/theopenlane/core/v2/internal/ent/generated",
		EntPackage:    "generated",
		ModelPackage:  "model",
		Entity:        entity,
		GroupByFields: fields,
	})

	assert.Contains(t, out, "case model.VulnerabilityGroupByFieldCveID: column = vulnerability.FieldCveID")
	assert.Contains(t, out, "case model.VulnerabilityGroupByFieldExternalOwnerID: column = vulnerability.FieldExternalOwnerID")
	assert.Contains(t, out, "filter, err = where.P()")
	assert.Contains(t, out, "!errors.Is(err, generated.ErrEmptyVulnerabilityWhereInput)")
	assert.Contains(t, out, "query = query.Where(filter)")
	assert.Contains(t, out, "groupby.NewQuery(column, filter, orderBy, after, first, before, last)")
	assert.Contains(t, out, "withTransactionalMutation(ctx).Vulnerability.Query()")
	assert.Contains(t, out, "query.Clone().Modify(groupQuery.Modifier).Scan(ctx, &rows)")
	assert.Contains(t, out, "query.Modify(groupQuery.TotalModifier).Int(ctx)")
	assert.NotContains(t, out, "archivedStatus")
}

func TestRenderGroupByTemplateArchivable(t *testing.T) {
	entity, fields := groupByResolverFields(groupByQueryField())

	out := renderTestTemplate(t, "groupby.gotpl", &crudResolver{
		Field:             groupByQueryField(),
		EntImport:         "github.com/theopenlane/core/v2/internal/ent/generated",
		EntPackage:        "generated",
		ArchivableSchemas: map[string]bool{"Vulnerability": true},
		Entity:            entity,
		GroupByFields:     fields,
	})

	assert.Contains(t, out, "where.StatusNEQ = &archivedStatus")
}
