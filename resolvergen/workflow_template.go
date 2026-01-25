package resolvergen

import (
	"bytes"
	"html/template"
	"strings"
)

type workflowResolverTemplate struct {
	HelperName string
	ObjectType string
	EntPackage string
	IsTimeline bool
}

func renderWorkflowTemplate(input *workflowResolverTemplate) string {
	t, err := template.New("workflow.gotpl").ParseFS(templates, "templates/workflow.gotpl")
	if err != nil {
		panic(err)
	}

	var code bytes.Buffer
	if err = t.Execute(&code, input); err != nil {
		panic(err)
	}

	return strings.Trim(code.String(), "\t \n")
}
