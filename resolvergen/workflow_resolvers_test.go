package resolvergen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// normalizeWhitespace collapses all whitespace sequences into single spaces
// and removes spaces around dots, parens, and brackets for consistent comparison
func normalizeWhitespace(s string) string {
	fields := strings.Fields(s)
	joined := strings.Join(fields, " ")
	joined = strings.ReplaceAll(joined, ". ", ".")
	joined = strings.ReplaceAll(joined, " .", ".")
	joined = strings.ReplaceAll(joined, "( ", "(")
	joined = strings.ReplaceAll(joined, " )", ")")
	joined = strings.ReplaceAll(joined, "[ ", "[")
	joined = strings.ReplaceAll(joined, " ]", "]")

	return joined
}

func TestUpdateWorkflowResolvers(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	graphDir := filepath.Join(root, "internal", "graphapi")
	if err := os.MkdirAll(graphDir, 0o755); err != nil { // nolint:mnd
		t.Fatalf("mkdir graphapi: %v", err)
	}

	source := `package graphapi

import (
	"context"

	"entgo.io/contrib/entgql"
	"example.com/test/internal/ent/generated"
)

func (r *controlResolver) HasPendingWorkflow(ctx context.Context, obj *generated.Control) (bool, error) {
	return workflowResolverHasPending(ctx, generated.TypeControl, obj.ID)
}

func (r *controlResolver) HasWorkflowHistory(ctx context.Context, obj *generated.Control) (bool, error) {
	return workflowResolverHasHistory(ctx, generated.TypeControl, obj.ID)
}

func (r *controlResolver) ActiveWorkflowInstances(ctx context.Context, obj *generated.Control) ([]*generated.WorkflowInstance, error) {
	return workflowResolverActiveInstances(ctx, generated.TypeControl, obj.ID)
}

func (r *controlResolver) WorkflowTimeline(ctx context.Context, obj *generated.Control, after *entgql.Cursor[string], first *int, before *entgql.Cursor[string], last *int, orderBy []*generated.WorkflowEventOrder, where *generated.WorkflowEventWhereInput, includeEmitFailures *bool) (*generated.WorkflowEventConnection, error) {
	return workflowResolverTimeline(ctx, generated.TypeControl, obj.ID, after, first, before, last, orderBy, where, includeEmitFailures)
}
`

	resolverPath := filepath.Join(graphDir, "control.resolvers.go")
	if err := os.WriteFile(resolverPath, []byte(source), 0o600); err != nil { // nolint:mnd
		t.Fatalf("write resolver file: %v", err)
	}

	original := source

	if err := UpdateWorkflowResolvers(graphDir); err != nil {
		t.Fatalf("UpdateWorkflowResolvers failed: %v", err)
	}

	updated, err := os.ReadFile(resolverPath)
	if err != nil {
		t.Fatalf("read updated resolver file: %v", err)
	}

	updatedStr := string(updated)
	if updatedStr != original {
		t.Fatalf("expected resolver file to remain unchanged")
	}

	helperPath := filepath.Join(graphDir, workflowResolverHelperFile)

	helperBytes, err := os.ReadFile(helperPath)
	if err != nil {
		t.Fatalf("read helper file: %v", err)
	}

	helperStr := string(helperBytes)
	if !strings.Contains(helperStr, "package graphapi") {
		t.Fatalf("expected helper file to use graphapi package")
	}

	if !strings.Contains(helperStr, "example.com/test/internal/ent/generated") {
		t.Fatalf("expected helper file to include generated import")
	}

	if !strings.Contains(helperStr, "example.com/test/common/enums") {
		t.Fatalf("expected helper file to include enums import")
	}

	if !strings.Contains(helperStr, "workflowResolverActiveInstance") {
		t.Fatalf("expected helper file to include active instance helper")
	}
}

func TestRenderWorkflowTemplate(t *testing.T) {
	t.Parallel()

	rendered := renderWorkflowTemplate(&workflowResolverTemplate{
		HelperName: "workflowResolverHasPending",
		ObjectType: "Control",
		EntPackage: "generated",
	})

	normalized := normalizeWhitespace(rendered)
	if !strings.Contains(normalized, "return workflowResolverHasPending(ctx, generated.TypeControl, obj.ID)") {
		t.Fatalf("expected template to render helper call")
	}

	renderedTimeline := renderWorkflowTemplate(&workflowResolverTemplate{
		HelperName: "workflowResolverTimeline",
		ObjectType: "Control",
		EntPackage: "generated",
		IsTimeline: true,
	})

	normalizedTimeline := normalizeWhitespace(renderedTimeline)
	timelineCall := "return workflowResolverTimeline(ctx, generated.TypeControl, obj.ID, after, first, before, last, orderBy, where, includeEmitFailures"
	if !strings.Contains(normalizedTimeline, timelineCall) {
		t.Fatalf("expected timeline template to render helper call with params")
	}
}
