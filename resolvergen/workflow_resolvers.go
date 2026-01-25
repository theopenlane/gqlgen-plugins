package resolvergen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// workflowResolverHelperFile is the filename for the generated workflow resolver helper functions
const workflowResolverHelperFile = "workflow_resolvers_gen.go"

// workflowResolverHelpers maps gqlgen-generated resolver method names to their helper function names.
var workflowResolverHelpers = map[string]string{
	"HasPendingWorkflow":      "workflowResolverHasPending",
	"HasWorkflowHistory":      "workflowResolverHasHistory",
	"ActiveWorkflowInstances": "workflowResolverActiveInstances",
	"WorkflowTimeline":        "workflowResolverTimeline",
}

// workflowResolverUpdate tracks the state of processing a single resolver file
type workflowResolverUpdate struct {
	// foundWorkflow indicates whether any workflow resolver methods were found in the file
	foundWorkflow bool
	// packageName is the Go package name extracted from the file
	packageName string
	// generatedImportPath is the import path for the ent generated package
	generatedImportPath string
	// graphCommonImportPath is the import path for the graphapi common package
	graphCommonImportPath string
}

// UpdateWorkflowResolvers generates shared workflow resolver helper implementations
// when workflow fields are present in resolver files.
func UpdateWorkflowResolvers(graphResolverDir string) error {
	if graphResolverDir == "" {
		return ErrGraphResolverDirRequired
	}

	resolverFiles, err := filepath.Glob(filepath.Join(graphResolverDir, "*.resolvers.go"))
	if err != nil {
		return fmt.Errorf("list resolver files: %w", err)
	}

	var (
		foundWorkflow         bool
		packageName           string
		generatedImportPath   string
		graphCommonImportPath string
	)

	for _, path := range resolverFiles {
		update, err := updateWorkflowResolverFile(path)
		if err != nil {
			return fmt.Errorf("update workflow resolvers in %s: %w", path, err)
		}

		if update.foundWorkflow {
			foundWorkflow = true
		}

		if packageName == "" && update.packageName != "" {
			packageName = update.packageName
		}

		if generatedImportPath == "" && update.generatedImportPath != "" {
			generatedImportPath = update.generatedImportPath
		}

		if graphCommonImportPath == "" && update.graphCommonImportPath != "" {
			graphCommonImportPath = update.graphCommonImportPath
		}
	}

	if !foundWorkflow {
		return nil
	}

	moduleRoot := moduleRootFromGenerated(generatedImportPath)
	if moduleRoot == "" {
		moduleRoot = moduleRootFromGoMod(graphResolverDir)
	}

	if moduleRoot == "" {
		return ErrModuleRootNotFound
	}

	if generatedImportPath == "" {
		generatedImportPath = filepath.ToSlash(filepath.Join(moduleRoot, "internal/ent/generated"))
	}

	if graphCommonImportPath == "" {
		graphCommonImportPath = filepath.ToSlash(filepath.Join(moduleRoot, "internal/graphapi/common"))
	}

	enumsImportPath := filepath.ToSlash(filepath.Join(moduleRoot, "common/enums"))

	if packageName == "" {
		packageName = "graphapi"
	}

	helperContent, err := renderWorkflowResolverHelpers(packageName, generatedImportPath, graphCommonImportPath, enumsImportPath)
	if err != nil {
		return err
	}

	helperPath := filepath.Join(graphResolverDir, workflowResolverHelperFile)
	if err := writeFileIfChanged(helperPath, helperContent); err != nil {
		return fmt.Errorf("write workflow resolver helpers: %w", err)
	}

	return nil
}

// updateWorkflowResolverFile parses a single resolver file and collects metadata
// about workflow resolver methods for helper generation.
func updateWorkflowResolverFile(path string) (workflowResolverUpdate, error) {
	update := workflowResolverUpdate{}
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return update, fmt.Errorf("parse file: %w", err)
	}

	update.packageName = file.Name.Name
	update.generatedImportPath = findGeneratedImportPath(file)
	update.graphCommonImportPath = findGraphCommonImportPath(file)

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name == nil || fn.Body == nil {
			continue
		}

		_, ok = workflowResolverHelpers[fn.Name.Name]
		if !ok {
			continue
		}

		update.foundWorkflow = true
	}

	return update, nil
}

// findGeneratedImportPath searches the file's imports for the ent generated package path.
// Returns the full import path if found, or empty string if not present.
func findGeneratedImportPath(file *ast.File) string {
	for _, spec := range file.Imports {
		path := strings.Trim(spec.Path.Value, "\"")
		if strings.HasSuffix(path, "/internal/ent/generated") {
			return path
		}
	}

	return ""
}

// findGraphCommonImportPath searches the file's imports for the graphapi common package path.
// Returns the full import path if found, or empty string if not present.
func findGraphCommonImportPath(file *ast.File) string {
	for _, spec := range file.Imports {
		path := strings.Trim(spec.Path.Value, "\"")
		if strings.HasSuffix(path, "/internal/graphapi/common") {
			return path
		}
	}

	return ""
}

// moduleRootFromGenerated extracts the module root from a generated package import path.
// Given "github.com/example/project/internal/ent/generated", returns "github.com/example/project".
func moduleRootFromGenerated(importPath string) string {
	const suffix = "/internal/ent/generated"
	if before, ok := strings.CutSuffix(importPath, suffix); ok {
		return before
	}

	return ""
}

// moduleRootFromGoMod walks up the directory tree from startDir looking for a go.mod file.
// If found, extracts and returns the module path. Returns empty string if no go.mod is found.
func moduleRootFromGoMod(startDir string) string {
	dir := startDir
	for {
		modPath := filepath.Join(dir, "go.mod")

		data, err := os.ReadFile(modPath)
		if err == nil {
			if modulePath := modulePathFromGoMod(data); modulePath != "" {
				return modulePath
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}

		dir = parent
	}
}

// modulePathFromGoMod parses go.mod file contents and extracts the module path from the module directive.
func modulePathFromGoMod(data []byte) string {
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(after)
		}
	}

	return ""
}

// renderWorkflowResolverHelpers generates the Go source code for the workflow resolver helper file.
// This file contains shared implementations for workflow resolvers that are called by the individual entity resolvers.
func renderWorkflowResolverHelpers(packageName, generatedImportPath, graphCommonImportPath, enumsImportPath string) ([]byte, error) {
	workflowEventImportPath := generatedImportPath + "/workflowevent"
	workflowInstanceImportPath := generatedImportPath + "/workflowinstance"
	workflowObjectRefImportPath := generatedImportPath + "/workflowobjectref"
	workflowProposalImportPath := generatedImportPath + "/workflowproposal"

	content := fmt.Sprintf(`// THIS CODE IS REGENERATED BY github.com/theopenlane/gqlgen-plugins. DO NOT EDIT.
package %s

import (
	"context"

	"entgo.io/contrib/entgql"
	"entgo.io/ent/dialect/sql"
	"%s"
	"%s"
	"%s"
	"%s"
	"%s"
	"%s"
	"%s"
)

// workflowResolverHasPending checks if the object has any pending workflow proposals (draft or submitted).
func workflowResolverHasPending(ctx context.Context, objectType string, objectID string) (bool, error) {
	if objectID == "" {
		return false, nil
	}

	wfType := enums.ToWorkflowObjectType(objectType)
	if wfType == nil {
		return false, nil
	}

	query := withTransactionalMutation(ctx).WorkflowObjectRef.Query()
	query = generated.ApplyWorkflowObjectRefObjectPredicate(query, *wfType, objectID)

	exists, err := query.Where(
		workflowobjectref.HasWorkflowProposalsWith(
			workflowproposal.StateIn(
				enums.WorkflowProposalStateDraft,
				enums.WorkflowProposalStateSubmitted,
			),
		),
	).Exist(ctx)
	if err != nil {
		return false, parseRequestError(ctx, err, common.Action{Action: common.ActionGet, Object: "workflowproposal"})
	}

	return exists, nil
}

// workflowResolverHasHistory checks if the object has any workflow history (completed or failed instances).
func workflowResolverHasHistory(ctx context.Context, objectType string, objectID string) (bool, error) {
	if objectID == "" {
		return false, nil
	}

	wfType := enums.ToWorkflowObjectType(objectType)
	if wfType == nil {
		return false, nil
	}

	query := withTransactionalMutation(ctx).WorkflowInstance.Query()
	query = generated.ApplyWorkflowInstanceObjectPredicate(query, *wfType, objectID)

	exists, err := query.Where(
		workflowinstance.StateIn(
			enums.WorkflowInstanceStateCompleted,
			enums.WorkflowInstanceStateFailed,
		),
	).Exist(ctx)
	if err != nil {
		return false, parseRequestError(ctx, err, common.Action{Action: common.ActionGet, Object: "workflowinstance"})
	}

	return exists, nil
}

// workflowResolverActiveInstances returns all active workflow instances (running or paused) for the object.
func workflowResolverActiveInstances(ctx context.Context, objectType string, objectID string) ([]*generated.WorkflowInstance, error) {
	query, err := workflowResolverInstanceQuery(ctx, objectType, objectID)
	if err != nil || query == nil {
		return nil, err
	}

	query = query.Where(
		workflowinstance.StateIn(
			enums.WorkflowInstanceStateRunning,
			enums.WorkflowInstanceStatePaused,
		),
	).Order(
		workflowinstance.ByLastEvaluatedAt(sql.OrderDesc()),
		workflowinstance.ByUpdatedAt(sql.OrderDesc()),
	)

	res, err := query.All(ctx)
	if err != nil {
		return nil, parseRequestError(ctx, err, common.Action{Action: common.ActionGet, Object: "workflowinstance"})
	}

	return res, nil
}

// workflowResolverTimeline returns the workflow event timeline for an object across all its workflow instances.
func workflowResolverTimeline(ctx context.Context, objectType string, objectID string, after *entgql.Cursor[string], first *int, before *entgql.Cursor[string], last *int, orderBy []*generated.WorkflowEventOrder, where *generated.WorkflowEventWhereInput, includeEmitFailures *bool) (*generated.WorkflowEventConnection, error) {
	if objectID == "" {
		return &generated.WorkflowEventConnection{}, nil
	}

	wfType := enums.ToWorkflowObjectType(objectType)
	if wfType == nil {
		return &generated.WorkflowEventConnection{}, nil
	}

	// Get all workflow instance IDs for this object
	instanceIDs, err := workflowResolverInstanceIDs(ctx, *wfType, objectID)
	if err != nil {
		return nil, err
	}

	if len(instanceIDs) == 0 {
		return &generated.WorkflowEventConnection{}, nil
	}

	// Set default ordering
	if orderBy == nil {
		orderBy = []*generated.WorkflowEventOrder{
			{
				Field:     generated.WorkflowEventOrderFieldCreatedAt,
				Direction: entgql.OrderDirectionAsc,
			},
		}
	}

	query, err := withTransactionalMutation(ctx).WorkflowEvent.Query().CollectFields(ctx)
	if err != nil {
		return nil, parseRequestError(ctx, err, common.Action{Action: common.ActionGet, Object: "workflowevent"})
	}

	if where == nil {
		where = &generated.WorkflowEventWhereInput{}
	}

	// Filter to events for this object's workflow instances
	query = query.Where(workflowevent.WorkflowInstanceIDIn(instanceIDs...))

	// Filter to timeline event types
	includeFailures := includeEmitFailures != nil && *includeEmitFailures
	timelineEventTypes := workflowTimelineEventTypes(includeFailures)
	query = query.Where(workflowevent.EventTypeIn(timelineEventTypes...))

	res, err := query.Paginate(
		ctx,
		after,
		first,
		before,
		last,
		generated.WithWorkflowEventOrder(orderBy),
		generated.WithWorkflowEventFilter(where.Filter),
	)
	if err != nil {
		return nil, parseRequestError(ctx, err, common.Action{Action: common.ActionGet, Object: "workflowevent"})
	}

	return res, nil
}

// workflowResolverInstanceIDs returns all workflow instance IDs for the given object.
func workflowResolverInstanceIDs(ctx context.Context, wfType enums.WorkflowObjectType, objectID string) ([]string, error) {
	query := withTransactionalMutation(ctx).WorkflowInstance.Query()
	query = generated.ApplyWorkflowInstanceObjectPredicate(query, wfType, objectID)

	return query.IDs(ctx)
}

// workflowResolverInstanceQuery builds a base query for workflow instances associated with the object.
func workflowResolverInstanceQuery(ctx context.Context, objectType string, objectID string) (*generated.WorkflowInstanceQuery, error) {
	if objectID == "" {
		return nil, nil
	}

	wfType := enums.ToWorkflowObjectType(objectType)
	if wfType == nil {
		return nil, nil
	}

	query, err := withTransactionalMutation(ctx).WorkflowInstance.Query().CollectFields(ctx)
	if err != nil {
		return nil, parseRequestError(ctx, err, common.Action{Action: common.ActionGet, Object: "workflowinstance"})
	}

	query = generated.ApplyWorkflowInstanceObjectPredicate(query, *wfType, objectID)

	return query, nil
}

// workflowTimelineEventTypes returns the event types included in timeline queries.
func workflowTimelineEventTypes(includeEmitFailures bool) []enums.WorkflowEventType {
	eventTypes := []enums.WorkflowEventType{
		enums.WorkflowEventTypeInstanceTriggered,
		enums.WorkflowEventTypeAssignmentCreated,
		enums.WorkflowEventTypeActionCompleted,
		enums.WorkflowEventTypeInstanceCompleted,
	}

	if includeEmitFailures {
		eventTypes = append(eventTypes,
			enums.WorkflowEventTypeEmitFailed,
			enums.WorkflowEventTypeEmitRecovered,
			enums.WorkflowEventTypeEmitFailedTerminal,
		)
	}

	return eventTypes
}
`, packageName, enumsImportPath, generatedImportPath, graphCommonImportPath, workflowEventImportPath, workflowInstanceImportPath, workflowObjectRefImportPath, workflowProposalImportPath)

	formatted, err := format.Source([]byte(content))
	if err != nil {
		return nil, fmt.Errorf("format workflow resolver helpers: %w", err)
	}

	return formatted, nil
}

// writeFileIfChanged writes content to a file only if the content differs from the existing file.
// This prevents unnecessary file modifications and timestamp changes during code generation.
func writeFileIfChanged(path string, content []byte) error {
	if existing, err := os.ReadFile(path); err == nil {
		if bytes.Equal(existing, content) {
			return nil
		}
	}

	return os.WriteFile(path, content, 0o600) // nolint:mnd
}
