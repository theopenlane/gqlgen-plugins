package resolvergen

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

const fakeCatalogSchema = `type Query {
  standards(after: Cursor, first: Int, before: Cursor, last: Int, orderBy: [StandardOrder!], where: StandardWhereInput): StandardConnection!
  controlObjectives(after: Cursor, first: Int, before: Cursor, last: Int, where: ControlObjectiveWhereInput): ControlObjectiveConnection!
}
extend type Query {
  standardSearch(query: String!, after: Cursor, first: Int, before: Cursor, last: Int): StandardConnection!
}
`

const expectedControlObjectiveCatalogSchema = `extend type Query {
    """
    controlObjectivesCatalog lists the externally visible system-owned controlObjectives available for adoption
    """
    controlObjectivesCatalog(after: Cursor, first: Int, before: Cursor, last: Int, where: ControlObjectiveWhereInput): ControlObjectiveConnection!
}
extend type Mutation {
    """
    adoptControlObjective copies the externally visible system-owned controlObjective into the organization, returning the organization's copy; idempotent per organization
    """
    adoptControlObjective(catalogID: ID!, input: CreateControlObjectiveInput): ControlObjectiveCreatePayload!
}
`

const expectedStandardCatalogSchema = `extend type Query {
    """
    standardsCatalog lists the externally visible system-owned standards available for adoption
    """
    standardsCatalog(after: Cursor, first: Int, before: Cursor, last: Int, orderBy: [StandardOrder!], where: StandardWhereInput): StandardConnection!
}
extend type Mutation {
    """
    adoptStandard copies the externally visible system-owned standard into the organization, returning the organization's copy; idempotent per organization
    """
    adoptStandard(catalogID: ID!, input: CreateStandardInput): StandardCreatePayload!
}
`

// renderCatalogTemplate renders a catalog template with stubs for the funcs gqlgen normally supplies
func renderCatalogTemplate(t *testing.T, name string, input *crudResolver) string {
	t.Helper()

	tmpl, err := template.New(name).Funcs(template.FuncMap{
		"getEntityName":          getEntityName,
		"toLower":                strings.ToLower,
		"hasArgument":            hasArgument,
		"hasStatusField":         func(entityName string) bool { return input.ArchivableSchemas[entityName] },
		"isCatalogSchema":        func(entityName string) bool { return input.CatalogSchemas[entityName] },
		"getArchivedStatusValue": getArchivedStatusEnum,
		"isListType":             isListType,
		"modelPackage":           modelPackage,
		"reserveImport":          func(...string) string { return "" },
	}).ParseFS(templates, "templates/"+name)
	require.NoError(t, err)

	var out bytes.Buffer

	require.NoError(t, tmpl.Execute(&out, input))

	return normalizeWhitespace(out.String())
}

func catalogListField() *codegen.Field {
	return &codegen.Field{
		FieldDefinition: &ast.FieldDefinition{
			Name: "standardsCatalog",
			Arguments: ast.ArgumentDefinitionList{
				{Name: "after", Type: ast.NamedType("Cursor", nil)},
				{Name: "first", Type: ast.NamedType("Int", nil)},
				{Name: "before", Type: ast.NamedType("Cursor", nil)},
				{Name: "last", Type: ast.NamedType("Int", nil)},
				{Name: "orderBy", Type: ast.ListType(ast.NonNullNamedType("StandardOrder", nil), nil)},
				{Name: "where", Type: ast.NamedType("StandardWhereInput", nil)},
			},
		},
		GoFieldName:   "StandardsCatalog",
		TypeReference: &config.TypeReference{Definition: &ast.Definition{Name: "StandardConnection"}},
		Object:        &codegen.Object{Definition: &ast.Definition{Name: "Query"}},
	}
}

func catalogAdoptField() *codegen.Field {
	return &codegen.Field{
		FieldDefinition: &ast.FieldDefinition{Name: "adoptStandard"},
		GoFieldName:     "AdoptStandard",
		TypeReference:   &config.TypeReference{Definition: &ast.Definition{Name: "StandardCreatePayload"}},
		Object:          &codegen.Object{Definition: &ast.Definition{Name: "Mutation"}},
	}
}

func TestWithCatalogSchemas(t *testing.T) {
	plugin := NewWithOptions(WithCatalogSchemas([]string{"standard", "control_objective"}))

	assert.True(t, plugin.catalogSchemas["Standard"])
	assert.True(t, plugin.catalogSchemas["ControlObjective"])
	assert.False(t, plugin.catalogSchemas["Program"])
}

func TestWriteCatalogSchema(t *testing.T) {
	t.Parallel()

	schemaDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(schemaDir, "ent.graphql"), []byte(fakeCatalogSchema), 0o600)) // nolint:mnd

	plugin := NewWithOptions(WithCatalogSchemas([]string{"standard", "control_objective"}))

	require.NoError(t, plugin.WriteCatalogSchema(schemaDir))

	content, err := os.ReadFile(filepath.Join(schemaDir, "standardcatalog.graphql"))
	require.NoError(t, err)
	assert.Equal(t, expectedStandardCatalogSchema, string(content))

	content, err = os.ReadFile(filepath.Join(schemaDir, "controlobjectivecatalog.graphql"))
	require.NoError(t, err)
	assert.Equal(t, expectedControlObjectiveCatalogSchema, string(content))

	require.NoError(t, plugin.WriteCatalogSchema(schemaDir))
}

func TestWriteCatalogSchemaErrors(t *testing.T) {
	t.Parallel()

	schemaDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(schemaDir, "ent.graphql"), []byte(fakeCatalogSchema), 0o600)) // nolint:mnd

	assert.ErrorIs(t, NewWithOptions(WithCatalogSchemas([]string{"standard"})).WriteCatalogSchema(""), ErrGraphSchemaDirRequired)
	assert.ErrorIs(t, NewWithOptions(WithCatalogSchemas([]string{"program"})).WriteCatalogSchema(schemaDir), ErrCatalogListQueryNotFound)

	require.NoError(t, New().WriteCatalogSchema(schemaDir))
	assert.NoFileExists(t, filepath.Join(schemaDir, "standardcatalog.graphql"))
}

func TestRenderCatalogListTemplate(t *testing.T) {
	rendered := renderCatalogTemplate(t, "catalog_list.gotpl", &crudResolver{
		Field:       catalogListField(),
		EntImport:   "github.com/example/internal/ent/generated",
		EntPackage:  "generated",
		RuleImport:  "github.com/example/internal/ent/privacy/rule",
		RulePackage: "rule",
	})

	assert.Contains(t, rendered, "first, last = graphutils.SetFirstLastDefaults(first, last, r.maxResultLimit)")
	assert.Contains(t, rendered, "orderBy = []*generated.StandardOrder{ { Field: generated.StandardOrderFieldCreatedAt, Direction: entgql.OrderDirectionDesc, }, }")
	assert.Contains(t, rendered, "ctx = rule.WithInternalContext(ctx)")
	assert.Contains(t, rendered, "query, err := withTransactionalMutation(ctx).Standard.Query().CollectFields(ctx)")
	assert.Contains(t, rendered, "query, err = where.Filter(query)")
	assert.Contains(t, rendered, "query = query.Where(standard.SystemOwned(true), standard.ExternallyVisible(true))")
	assert.Contains(t, rendered, "res, err := query.Paginate(ctx, after, first, before, last, generated.WithStandardOrder(orderBy),)")
	assert.NotContains(t, rendered, "WithStandardFilter")
	assert.Contains(t, rendered, "return res, nil")
}

func TestRenderCatalogAdoptTemplate(t *testing.T) {
	rendered := renderCatalogTemplate(t, "catalog_adopt.gotpl", &crudResolver{
		Field:            catalogAdoptField(),
		ModelPackage:     "model",
		EntImport:        "github.com/example/internal/ent/generated",
		EntPackage:       "generated",
		EntityOpsImport:  "github.com/example/internal/ent/entityops",
		EntityOpsPackage: "entityops",
		JSONXImport:      "github.com/example/pkg/jsonx",
		JSONXPackage:     "jsonx",
	})

	assert.Contains(t, rendered, "ctx, err := common.SetOrganizationInAuthContext(ctx, nil)")
	assert.Contains(t, rendered, "orgID, err := auth.GetOrganizationIDFromContext(ctx)")
	assert.Contains(t, rendered, "overlay, err := jsonx.ToRawMessage(input)")
	assert.Contains(t, rendered, "id, _, err := entityops.SchemaStandard.Adopt(ctx, withTransactionalMutation(ctx), catalogID, orgID, overlay)")
	assert.Contains(t, rendered, "case errors.Is(err, entityops.ErrCatalogRowNotSystemOwned), errors.Is(err, entityops.ErrCatalogRowNotVisible): return nil, common.NewNotFoundError(\"standard\")")
	assert.Contains(t, rendered, "case err != nil: return nil, parseRequestError(ctx, err, common.Action{Action: common.ActionCreate, Object: \"standard\"})")
	assert.Contains(t, rendered, "res, err := withTransactionalMutation(ctx).Standard.Get(ctx, id)")
	assert.Contains(t, rendered, "return &model.StandardCreatePayload{ Standard: res, }, nil")
}

func TestIsCatalogField(t *testing.T) {
	plugin := NewWithOptions(WithCatalogSchemas([]string{"standard"}))

	assert.True(t, plugin.isCatalogQuery(catalogListField()))
	assert.False(t, plugin.isAdoptMutation(catalogListField()))
	assert.True(t, plugin.isAdoptMutation(catalogAdoptField()))
	assert.False(t, plugin.isCatalogQuery(catalogAdoptField()))

	other := New()

	assert.False(t, other.isCatalogQuery(catalogListField()))
	assert.False(t, other.isAdoptMutation(catalogAdoptField()))
}

func TestRenderListTemplateCatalogDefaultFilter(t *testing.T) {
	field := catalogListField()
	field.FieldDefinition.Name = "standards"
	field.GoFieldName = "Standards"

	rendered := renderCatalogTemplate(t, "list.gotpl", &crudResolver{
		Field:          field,
		EntImport:      "github.com/example/internal/ent/generated",
		EntPackage:     "generated",
		CatalogSchemas: map[string]bool{"Standard": true},
	})

	assert.Contains(t, rendered, "if where.SystemOwned == nil && where.SystemOwnedNEQ == nil { systemOwned := false where.SystemOwned = &systemOwned }")

	plain := renderCatalogTemplate(t, "list.gotpl", &crudResolver{
		Field:      field,
		EntImport:  "github.com/example/internal/ent/generated",
		EntPackage: "generated",
	})

	assert.NotContains(t, plain, "where.SystemOwned")
}
