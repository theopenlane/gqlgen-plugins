package graphutils

import (
	"strconv"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestSetFirstLastDefaults(t *testing.T) {
	maxPageSize := 10

	tests := []struct {
		name          string
		first         *int
		last          *int
		expectedFirst *int
		expectedLast  *int
	}{
		{
			name:          "both nil",
			first:         nil,
			last:          nil,
			expectedFirst: &maxPageSize,
			expectedLast:  nil,
		},
		{
			name:          "first greater than maxPageSize",
			first:         lo.ToPtr(15),
			last:          nil,
			expectedFirst: &maxPageSize,
			expectedLast:  nil,
		},
		{
			name:          "last greater than maxPageSize",
			first:         nil,
			last:          lo.ToPtr(15),
			expectedFirst: nil,
			expectedLast:  &maxPageSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, last := SetFirstLastDefaults(tt.first, tt.last, &maxPageSize)

			assert.Equal(t, tt.expectedFirst, first, "expected first: %v, got: %v", tt.expectedFirst, first)

			assert.Equal(t, tt.expectedLast, last, "expected last: %v, got: %v", tt.expectedLast, last)
		})
	}
}
func TestSetDefaultPaginationLimit(t *testing.T) {
	maxPageSize := 10
	firstArg := "first"
	lastArg := "last"

	tests := []struct {
		name          string
		column        *graphql.CollectedField
		expectedFirst *ast.Value
		expectedLast  *ast.Value
	}{
		{
			name: "no arguments set",
			column: &graphql.CollectedField{
				Field: &ast.Field{
					Definition: &ast.FieldDefinition{
						Arguments: ast.ArgumentDefinitionList{
							&ast.ArgumentDefinition{Name: firstArg},
							&ast.ArgumentDefinition{Name: lastArg},
						},
					},
					Arguments: ast.ArgumentList{},
				},
			},
			expectedFirst: &ast.Value{
				Raw:  strconv.Itoa(maxPageSize),
				Kind: ast.IntValue,
			},
			expectedLast: nil,
		},
		{
			name: "first argument set within limit",
			column: &graphql.CollectedField{
				Field: &ast.Field{
					Definition: &ast.FieldDefinition{
						Arguments: ast.ArgumentDefinitionList{
							&ast.ArgumentDefinition{Name: firstArg},
							&ast.ArgumentDefinition{Name: lastArg},
						},
					},
					Arguments: ast.ArgumentList{
						&ast.Argument{
							Name: firstArg,
							Value: &ast.Value{
								Raw:  "5",
								Kind: ast.IntValue,
							},
						},
					},
				},
			},
			expectedFirst: &ast.Value{
				Raw:  "5",
				Kind: ast.IntValue,
			},
			expectedLast: nil,
		},
		{
			name: "first argument set beyond limit",
			column: &graphql.CollectedField{
				Field: &ast.Field{
					Definition: &ast.FieldDefinition{
						Arguments: ast.ArgumentDefinitionList{
							&ast.ArgumentDefinition{Name: firstArg},
							&ast.ArgumentDefinition{Name: lastArg},
						},
					},
					Arguments: ast.ArgumentList{
						&ast.Argument{
							Name: firstArg,
							Value: &ast.Value{
								Raw:  "15",
								Kind: ast.IntValue,
							},
						},
					},
				},
			},
			expectedFirst: &ast.Value{
				Raw:  strconv.Itoa(maxPageSize),
				Kind: ast.IntValue,
			},
			expectedLast: nil,
		},
		{
			name: "last argument set within limit",
			column: &graphql.CollectedField{
				Field: &ast.Field{
					Definition: &ast.FieldDefinition{
						Arguments: ast.ArgumentDefinitionList{
							&ast.ArgumentDefinition{Name: firstArg},
							&ast.ArgumentDefinition{Name: lastArg},
						},
					},
					Arguments: ast.ArgumentList{
						&ast.Argument{
							Name: lastArg,
							Value: &ast.Value{
								Raw:  "8",
								Kind: ast.IntValue,
							},
						},
					},
				},
			},
			expectedFirst: nil,
			expectedLast: &ast.Value{
				Raw:  "8",
				Kind: ast.IntValue,
			},
		},
		{
			name: "last argument set beyond limit",
			column: &graphql.CollectedField{
				Field: &ast.Field{
					Definition: &ast.FieldDefinition{
						Arguments: ast.ArgumentDefinitionList{
							&ast.ArgumentDefinition{Name: firstArg},
							&ast.ArgumentDefinition{Name: lastArg},
						},
					},
					Arguments: ast.ArgumentList{
						&ast.Argument{
							Name: lastArg,
							Value: &ast.Value{
								Raw:  "20",
								Kind: ast.IntValue,
							},
						},
					},
				},
			},
			expectedFirst: nil,
			expectedLast: &ast.Value{
				Raw:  strconv.Itoa(maxPageSize),
				Kind: ast.IntValue,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setDefaultPaginationLimit(tt.column, &maxPageSize)

			first := tt.column.Arguments.ForName(firstArg)
			last := tt.column.Arguments.ForName(lastArg)

			if tt.expectedFirst != nil {
				assert.NotNil(t, first)
				assert.Equal(t, tt.expectedFirst.Raw, first.Value.Raw)
			} else {
				assert.Nil(t, first)
			}

			if tt.expectedLast != nil {
				assert.NotNil(t, last)
				assert.Equal(t, tt.expectedLast.Raw, last.Value.Raw)
			} else {
				assert.Nil(t, last)
			}
		})
	}
}
