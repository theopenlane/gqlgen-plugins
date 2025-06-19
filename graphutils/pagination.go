package graphutils

import (
	"strconv"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
)

// setDefaultPaginationLimit sets the default pagination limit for the given column
func setDefaultPaginationLimit(column *graphql.CollectedField, maxPageSize *int) {
	defaultFirstValue := &ast.Value{
		Raw:  strconv.Itoa(*maxPageSize),
		Kind: ast.IntValue,
	}

	// make sure the args are there on the field
	first := column.Definition.Arguments.ForName(firstArg)
	if first != nil {
		// check to see if they are set as the arguments
		first := column.Arguments.ForName(firstArg)
		last := column.Arguments.ForName(lastArg)

		if first == nil && last == nil {
			column.Arguments = append(column.Arguments, &ast.Argument{
				Name:  firstArg,
				Value: defaultFirstValue,
			})

			return
		}

		if first != nil && first.Value != nil && first.Value.Raw != "" {
			setValue, err := strconv.Atoi(first.Value.Raw)
			if err == nil {
				if setValue <= *maxPageSize {
					return
				}

				first.Value = defaultFirstValue

				return
			}
		}

		if last != nil && last.Value != nil && last.Value.Raw != "" {
			setValue, err := strconv.Atoi(last.Value.Raw)
			if err == nil {
				if setValue <= *maxPageSize {
					return
				}

				last.Value = defaultFirstValue

				return
			}
		}

		column.Arguments = append(column.Arguments, &ast.Argument{
			Name:  firstArg,
			Value: defaultFirstValue,
		})
	}
}

// SetFirstLastDefaults sets the first and last values to the default limit if they are greater than the default limit
// if both are nil, return the default limit
func SetFirstLastDefaults(first, last, maxPageSize *int) (*int, *int) {
	// if both are nil, return the default limit
	if first == nil && last == nil {
		return maxPageSize, nil
	}

	// if first is greater than the default limit, set it to the default limit
	if first != nil {
		if *first > *maxPageSize {
			first = maxPageSize
		}
	}

	// if last is greater than the default limit, set it to the default limit
	if last != nil {
		if *last > *maxPageSize {
			last = maxPageSize
		}
	}

	return first, last
}
