package searchgen

import (
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theopenlane/entx/genhooks"
)

// renderHelpers renders the helper template with stubs for the funcs gqlgen
// normally supplies, so the gating logic can be asserted without a full codegen run
func renderHelpers(t *testing.T, data SearchResolverBuild) string {
	t.Helper()

	tmpl := template.New("helpers").Funcs(template.FuncMap{
		"toLower":       strings.ToLower,
		"toSnakeCase":   func(s string) string { return s },
		"toPlural":      func(s string) string { return s + "s" },
		"isIDField":     isIDField,
		"reserveImport": func(...string) string { return "" },
		"add":           func(a, b int) int { return a + b },
	})

	tmpl, err := tmpl.Parse(helperTemplate)
	require.NoError(t, err)

	var out strings.Builder
	require.NoError(t, tmpl.Execute(&out, data))

	return out.String()
}

func TestHelperTemplateAdminSearch(t *testing.T) {
	data := SearchResolverBuild{
		EntImport: "github.com/theopenlane/core/internal/ent/generated",
		IDFields:  defaultIDFields,
		Objects: []Object{
			{
				Name:        "Task",
				Fields:      []genhooks.Field{{Name: "Title", Type: "string"}},
				AdminFields: []genhooks.Field{{Name: "Title", Type: "string"}, {Name: "OwnerID", Type: "string"}},
			},
		},
	}

	t.Run("admin search excluded", func(t *testing.T) {
		data.IncludeAdminSearch = false
		out := renderHelpers(t, data)

		assert.Contains(t, out, "func searchTasks(")
		assert.NotContains(t, out, "adminSearch")
	})

	t.Run("admin search included", func(t *testing.T) {
		data.IncludeAdminSearch = true
		out := renderHelpers(t, data)

		assert.Contains(t, out, "func searchTasks(")
		assert.Contains(t, out, "func adminSearchTasks(")
		assert.Contains(t, out, "task.OwnerIDContainsFold(query)")
	})
}
