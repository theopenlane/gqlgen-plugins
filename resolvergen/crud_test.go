package resolvergen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestHasArgument(t *testing.T) {
	args := ast.ArgumentDefinitionList{
		{Name: "where"},
		{Name: "here"},
	}

	testCases := []struct {
		name     string
		argName  string
		expected bool
	}{
		{
			name:     "arg found",
			argName:  "where",
			expected: true,
		},
		{
			name:     "arg not found",
			argName:  "nowhere",
			expected: false,
		},
		{
			name:     "empty arg",
			argName:  "",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run("Get "+tc.name, func(t *testing.T) {
			res := hasArgument(tc.argName, args)
			assert.Equal(t, tc.expected, res)
		})
	}
}

func TestGetEntityName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "strip Create",
			input:    "CreateUser",
			expected: "User",
		},
		{
			name:     "strip Update",
			input:    "UpdatePost",
			expected: "Post",
		},
		{
			name:     "strip Delete",
			input:    "DeleteComment",
			expected: "Comment",
		},
		{
			name:     "strip Bulk",
			input:    "BulkUpdateProduct",
			expected: "Product",
		},
		{
			name:     "strip CSV + Bulk",
			input:    "BulkCSVOrder",
			expected: "Order",
		},
		{
			name:     "strip Connection",
			input:    "UserConnection",
			expected: "User",
		},
		{
			name:     "strip Payload",
			input:    "PayloadUser",
			expected: "User",
		},
		{
			name:     "no strip",
			input:    "User",
			expected: "User",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := getEntityName(tc.input)
			assert.Equal(t, tc.expected, res)
		})
	}
}
func TestIsCommentUpdateOnObject(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "is comment update",
			input:    "UpdateComment",
			expected: true,
		},
		{
			name:     "update task comment",
			input:    "UpdateTaskComment",
			expected: true,
		},
		{
			name:     "is not comment update",
			input:    "UpdatePost",
			expected: false,
		},
		{
			name:     "contains comment but not update",
			input:    "CreateComment",
			expected: false,
		},
		{
			name:     "contains update but not comment",
			input:    "UpdateUser",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := isCommentUpdateOnObject(tc.input)
			assert.Equal(t, tc.expected, res)
		})
	}
}
func TestGetInputObjectName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "strip Create and InputObject",
			input:    "CreateProductInput",
			expected: "Product",
		},
		{
			name:     "strip Update and InputObject",
			input:    "UpdateOrderInput",
			expected: "Order",
		},
		{
			name:     "strip InputObject",
			input:    "UserInput",
			expected: "User",
		},
		{
			name:     "no strip",
			input:    "User",
			expected: "User",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := getInputObjectName(tc.input)
			assert.Equal(t, tc.expected, res)
		})
	}
}
