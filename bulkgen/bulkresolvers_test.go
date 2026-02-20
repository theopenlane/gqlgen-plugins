package bulkgen

import (
	"os"
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
			WithCSVFieldMappingsFile("/tmp/csv_field_mappings.json"),
		)

		assert.Equal(t, "github.com/example/model", plugin.ModelPackage)
		assert.Equal(t, "github.com/example/ent/generated", plugin.EntGeneratedPackage)
		assert.Equal(t, "github.com/example/graphql", plugin.GraphQLImport)
		assert.Equal(t, "/tmp/csv", plugin.CSVOutputPath)
		assert.Equal(t, "github.com/example/csvgenerated", plugin.CSVGeneratedPackage)
		assert.Equal(t, "/tmp/csv_field_mappings.json", plugin.CSVFieldMappingsFile)
	})

	t.Run("no options", func(t *testing.T) {
		plugin := NewWithOptions()
		assert.Empty(t, plugin.ModelPackage)
		assert.Empty(t, plugin.EntGeneratedPackage)
		assert.Empty(t, plugin.CSVGeneratedPackage)
		assert.Empty(t, plugin.CSVFieldMappingsFile)
	})
}

func TestPluginName(t *testing.T) {
	plugin := New()
	assert.Equal(t, "bulkgen", plugin.Name())
}

func TestWithCSVFieldMappingsFile(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "standard path",
			path:     "/path/to/csv_field_mappings.json",
			expected: "/path/to/csv_field_mappings.json",
		},
		{
			name:     "empty string",
			path:     "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plugin := &Plugin{}
			opt := WithCSVFieldMappingsFile(tc.path)
			opt(plugin)
			assert.Equal(t, tc.expected, plugin.CSVFieldMappingsFile)
		})
	}
}

func TestLoadCSVFieldMappings(t *testing.T) {
	t.Run("returns nil for empty path", func(t *testing.T) {
		result := loadCSVFieldMappings("")
		assert.Nil(t, result)
	})

	t.Run("returns nil for non-existent file", func(t *testing.T) {
		result := loadCSVFieldMappings("/non/existent/path.json")
		assert.Nil(t, result)
	})

	t.Run("loads valid JSON file", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := tempDir + "/csv_field_mappings.json"

		content := `{
			"ActionPlan": [
				{"csvColumn": "AssignedToUserEmail", "targetField": "AssignedToUserID", "isSlice": false},
				{"csvColumn": "BlockedGroupNames", "targetField": "BlockedGroupIds", "isSlice": true}
			],
			"Control": [
				{"csvColumn": "PlatformNames", "targetField": "PlatformIds", "isSlice": true}
			]
		}`

		err := os.WriteFile(filePath, []byte(content), 0600)
		assert.NoError(t, err)

		result := loadCSVFieldMappings(filePath)
		assert.NotNil(t, result)
		assert.Len(t, result, 2)
		assert.Len(t, result["ActionPlan"], 2)
		assert.Equal(t, "AssignedToUserEmail", result["ActionPlan"][0].CSVColumn)
		assert.Equal(t, "AssignedToUserID", result["ActionPlan"][0].TargetField)
		assert.False(t, result["ActionPlan"][0].IsSlice)
		assert.True(t, result["ActionPlan"][1].IsSlice)
	})

	t.Run("returns nil for invalid JSON", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := tempDir + "/invalid.json"

		err := os.WriteFile(filePath, []byte("not valid json"), 0600)
		assert.NoError(t, err)

		result := loadCSVFieldMappings(filePath)
		assert.Nil(t, result)
	})
}

func TestGenerateSampleCSVWithCustomColumns(t *testing.T) {
	tempDir := t.TempDir()

	object := Object{
		Name:          "ActionPlan",
		Fields:        []string{"Name", "Description", "Status"},
		OperationType: "create",
		CSVFieldMappings: []CSVFieldMapping{
			{CSVColumn: "AssignedToUserEmail", TargetField: "AssignedToUserID", IsSlice: false},
			{CSVColumn: "BlockedGroupNames", TargetField: "BlockedGroupIds", IsSlice: true},
		},
	}

	err := generateSampleCSV(object, tempDir)
	assert.NoError(t, err)

	content, err := os.ReadFile(tempDir + "/sample_actionplan.csv")
	assert.NoError(t, err)

	contentStr := string(content)

	assert.Contains(t, contentStr, "Name,Description,Status,AssignedToUserEmail,BlockedGroupNames")
	assert.Contains(t, contentStr, "example_name,example_description,example_status")
	assert.Contains(t, contentStr, "example_assignedtouseremail")
	assert.Contains(t, contentStr, "example_blockedgroupnames1,example_blockedgroupnames2")
}

func TestPluralFieldName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "faq",
			input:    "TrustCenterFAQ",
			expected: "TrustCenterFAQs",
		},
		{
			name:     "trustcenter subprocessor",
			input:    "TrustCenterSubprocessor",
			expected: "TrustCenterSubprocessors",
		},
		{
			name:     "policy",
			input:    "InternalPolicy",
			expected: "InternalPolicies",
		},
		{
			name:     "control",
			input:    "Control",
			expected: "Controls",
		},
		{
			name:     "ActionPlanStatus",
			input:    "ActionPlanStatus",
			expected: "ActionPlanStatuses",
		},
		{
			name:     "user",
			input:    "User",
			expected: "Users",
		},
		{
			name:     "acronym in middle",
			input:    "APIToken",
			expected: "APITokens",
		},
		{
			name:     "Entity",
			input:    "Entity",
			expected: "Entities",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := pluralFieldName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGenerateSampleCSVWithoutCustomColumns(t *testing.T) {
	tempDir := t.TempDir()

	object := Object{
		Name:             "Simple",
		Fields:           []string{"Name", "Value"},
		OperationType:    "create",
		CSVFieldMappings: nil,
	}

	err := generateSampleCSV(object, tempDir)
	assert.NoError(t, err)

	content, err := os.ReadFile(tempDir + "/sample_simple.csv")
	assert.NoError(t, err)

	contentStr := string(content)

	assert.Contains(t, contentStr, "Name,Value")
	assert.Contains(t, contentStr, "example_name,example_value")
}
