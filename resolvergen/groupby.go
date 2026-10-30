package resolvergen

import (
	"strings"

	"entgo.io/ent/entc/gen"
	"github.com/99designs/gqlgen/codegen"
	gqltemplates "github.com/99designs/gqlgen/codegen/templates"
)

// entPascal is the naming ent uses for generated field constants
var entPascal = gen.Funcs["pascal"].(func(string) string)

// groupByResolverField maps a group by enum value to its ent field constant
type groupByResolverField struct {
	// EnumConstant is the gqlgen model enum constant, e.g. VulnerabilityGroupByFieldCveID
	EnumConstant string
	// Constant is the ent field constant, e.g. FieldCveID
	Constant string
}

// isGroupByQuery returns true if the field is a generated group by query
func isGroupByQuery(f *codegen.Field) bool {
	return isQuery(f) && strings.HasSuffix(f.GoFieldName, GroupByOperation) && strings.HasSuffix(f.TypeReference.Definition.Name, groupByConnectionSuffix)
}

// isGroupByRowsField returns true if the field is a group by bucket's nested connection of the grouped rows
func isGroupByRowsField(f *codegen.Field) bool {
	return f.Object != nil && strings.HasSuffix(f.Object.Name, groupByBucketSuffix) && strings.HasSuffix(f.TypeReference.Definition.Name, Connection)
}

// renderGroupBy renders the group by template
func (r *ResolverPlugin) renderGroupBy(f *codegen.Field) string {
	entity, fields := groupByResolverFields(f)

	return renderTemplate("groupby.gotpl", &crudResolver{
		Field:             f,
		EntImport:         r.entGeneratedPackage,
		EntPackage:        getEntPackageFromImport(r.entGeneratedPackage),
		GraphQLImport:     r.graphqlImport,
		ModelPackage:      r.modelPackage,
		ArchivableSchemas: r.archivableSchemas,
		Entity:            entity,
		GroupByFields:     fields,
	}, []string{filterTemplates})
}

// groupByResolverFields returns the grouped schema name and its enum values from the field argument
func groupByResolverFields(f *codegen.Field) (string, []groupByResolverField) {
	for _, arg := range f.Args {
		if arg.Name != groupByFieldArg {
			continue
		}

		enum := arg.TypeReference.Definition
		fields := make([]groupByResolverField, 0, len(enum.EnumValues))

		for _, v := range enum.EnumValues {
			fields = append(fields, groupByResolverField{
				EnumConstant: gqltemplates.ToGoModelName(enum.Name, v.Name),
				Constant:     "Field" + entPascal(strings.ToLower(v.Name)),
			})
		}

		return strings.TrimSuffix(enum.Name, groupByFieldEnumSuffix), fields
	}

	return "", nil
}
