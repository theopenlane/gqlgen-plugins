package resolvergen

import (
	"testing"

	"github.com/99designs/gqlgen/codegen"
	"github.com/stretchr/testify/assert"
)

func TestWithCSVGeneratedPackage(t *testing.T) {
	testCases := []struct {
		name     string
		pkg      string
		expected string
	}{
		{
			name:     "standard package path",
			pkg:      "github.com/example/project/csvgenerated",
			expected: "github.com/example/project/csvgenerated",
		},
		{
			name:     "empty string",
			pkg:      "",
			expected: "",
		},
		{
			name:     "simple package name",
			pkg:      "csvgenerated",
			expected: "csvgenerated",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plugin := NewWithOptions(WithCSVGeneratedPackage(tc.pkg))
			assert.Equal(t, tc.expected, plugin.csvGeneratedPackage)
		})
	}
}

func TestWithForceRegenerateBulkResolvers(t *testing.T) {
	testCases := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{
			name:     "enabled",
			enabled:  true,
			expected: true,
		},
		{
			name:     "disabled",
			enabled:  false,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plugin := NewWithOptions(WithForceRegenerateBulkResolvers(tc.enabled))
			assert.Equal(t, tc.expected, plugin.forceRegenerateBulkResolvers)
		})
	}
}

func TestShouldRegenerateBulkResolver(t *testing.T) {
	testCases := []struct {
		name          string
		fieldName     string
		forceEnabled  bool
		expectedRegen bool
	}{
		{
			name:          "CreateBulk with force enabled",
			fieldName:     "CreateBulkControl",
			forceEnabled:  true,
			expectedRegen: true,
		},
		{
			name:          "UpdateBulk with force enabled",
			fieldName:     "UpdateBulkPolicy",
			forceEnabled:  true,
			expectedRegen: true,
		},
		{
			name:          "DeleteBulk with force enabled",
			fieldName:     "DeleteBulkUser",
			forceEnabled:  true,
			expectedRegen: true,
		},
		{
			name:          "CreateBulk with force disabled",
			fieldName:     "CreateBulkControl",
			forceEnabled:  false,
			expectedRegen: false,
		},
		{
			name:          "non-bulk field with force enabled",
			fieldName:     "CreateUser",
			forceEnabled:  true,
			expectedRegen: false,
		},
		{
			name:          "field containing Bulk but not prefix",
			fieldName:     "SomeBulkOperation",
			forceEnabled:  true,
			expectedRegen: false,
		},
		{
			name:          "CSV field not matching bulk pattern",
			fieldName:     "UploadCSVControl",
			forceEnabled:  true,
			expectedRegen: false,
		},
		{
			name:          "empty field name with force enabled",
			fieldName:     "",
			forceEnabled:  true,
			expectedRegen: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plugin := NewWithOptions(WithForceRegenerateBulkResolvers(tc.forceEnabled))

			field := &codegen.Field{
				GoFieldName: tc.fieldName,
			}

			result := plugin.shouldRegenerateBulkResolver(field)
			assert.Equal(t, tc.expectedRegen, result)
		})
	}
}

func TestShouldRegenerateBulkResolverNilField(t *testing.T) {
	plugin := NewWithOptions(WithForceRegenerateBulkResolvers(true))
	result := plugin.shouldRegenerateBulkResolver(nil)
	assert.False(t, result)
}

func TestNewWithMultipleOptions(t *testing.T) {
	plugin := NewWithOptions(
		WithEntGeneratedPackage("github.com/example/ent/generated"),
		WithGraphQLImport("github.com/example/graphql"),
		WithCSVGeneratedPackage("github.com/example/csvgenerated"),
		WithForceRegenerateBulkResolvers(true),
		WithArchivableSchemas([]string{"control", "policy"}),
	)

	assert.Equal(t, "github.com/example/ent/generated", plugin.entGeneratedPackage)
	assert.Equal(t, "github.com/example/graphql", plugin.graphqlImport)
	assert.Equal(t, "github.com/example/csvgenerated", plugin.csvGeneratedPackage)
	assert.True(t, plugin.forceRegenerateBulkResolvers)
	assert.True(t, plugin.archivableSchemas["Control"])
	assert.True(t, plugin.archivableSchemas["Policy"])
}

func TestNewDefaults(t *testing.T) {
	plugin := New()
	assert.True(t, plugin.includeCustomFields)
	assert.False(t, plugin.forceRegenerateBulkResolvers)
	assert.Empty(t, plugin.csvGeneratedPackage)
}

func TestPluginName(t *testing.T) {
	plugin := New()
	assert.Equal(t, "resolvergen", plugin.Name())
}
