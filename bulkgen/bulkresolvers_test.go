package bulkgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractObjectNameFromCSVMutation(t *testing.T) {
	testCases := []struct {
		name         string
		mutationName string
		expected     string
	}{
		{
			name:         "updateBulkCSV prefix",
			mutationName: "updateBulkCSVControl",
			expected:     "Control",
		},
		{
			name:         "updateBulkCSV prefix with longer name",
			mutationName: "updateBulkCSVRiskAssessment",
			expected:     "RiskAssessment",
		},
		{
			name:         "bulkCSVUpdate prefix",
			mutationName: "bulkCSVUpdatePolicy",
			expected:     "Policy",
		},
		{
			name:         "csvBulkUpdate prefix",
			mutationName: "csvBulkUpdateUser",
			expected:     "User",
		},
		{
			name:         "createBulkCSV prefix",
			mutationName: "createBulkCSVControl",
			expected:     "Control",
		},
		{
			name:         "bulkCSVCreate prefix",
			mutationName: "bulkCSVCreatePolicy",
			expected:     "Policy",
		},
		{
			name:         "csvBulkCreate prefix",
			mutationName: "csvBulkCreateUser",
			expected:     "User",
		},
		{
			name:         "case insensitive matching",
			mutationName: "updatebulkcsvcontrol",
			expected:     "control",
		},
		{
			name:         "mixed case CSV variation",
			mutationName: "CreateBulkCsvControl",
			expected:     "Control",
		},
		{
			name:         "no matching prefix",
			mutationName: "createUser",
			expected:     "",
		},
		{
			name:         "bulk but not CSV",
			mutationName: "createBulkControl",
			expected:     "",
		},
		{
			name:         "empty string",
			mutationName: "",
			expected:     "",
		},
		{
			name:         "pattern in middle of name returns empty",
			mutationName: "doSomeUpdateBulkCSVProcessing",
			expected:     "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractObjectNameFromCSVMutation(tc.mutationName)
			assert.Equal(t, tc.expected, result)
		})
	}
}

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
			plugin := &Plugin{}
			opt := WithCSVGeneratedPackage(tc.pkg)
			opt(plugin)
			assert.Equal(t, tc.expected, plugin.CSVGeneratedPackage)
		})
	}
}

func TestNewWithOptions(t *testing.T) {
	t.Run("multiple options applied", func(t *testing.T) {
		plugin := NewWithOptions(
			WithModelPackage("github.com/example/model"),
			WithEntGeneratedPackage("github.com/example/ent/generated"),
			WithGraphQLImport("github.com/example/graphql"),
			WithCSVOutputPath("/tmp/csv"),
			WithCSVGeneratedPackage("github.com/example/csvgenerated"),
		)

		assert.Equal(t, "github.com/example/model", plugin.ModelPackage)
		assert.Equal(t, "github.com/example/ent/generated", plugin.EntGeneratedPackage)
		assert.Equal(t, "github.com/example/graphql", plugin.GraphQLImport)
		assert.Equal(t, "/tmp/csv", plugin.CSVOutputPath)
		assert.Equal(t, "github.com/example/csvgenerated", plugin.CSVGeneratedPackage)
	})

	t.Run("no options", func(t *testing.T) {
		plugin := NewWithOptions()
		assert.Empty(t, plugin.ModelPackage)
		assert.Empty(t, plugin.EntGeneratedPackage)
		assert.Empty(t, plugin.CSVGeneratedPackage)
	})
}

func TestPluginName(t *testing.T) {
	plugin := New()
	assert.Equal(t, "bulkgen", plugin.Name())
}
