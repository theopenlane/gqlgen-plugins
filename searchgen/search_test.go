package searchgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsIDField(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expected bool
	}{
		{
			name:     "ID field",
			field:    "ID",
			expected: true,
		},
		{
			name:     "DisplayID field",
			field:    "DisplayID",
			expected: true,
		},
		{
			name:     "Non-ID field",
			field:    "Name",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isIDField(tt.field, defaultIDFields)
			assert.Equal(t, tt.expected, result)
		})
	}
}
