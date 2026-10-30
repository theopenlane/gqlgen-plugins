package groupby

import (
	"entgo.io/ent/schema"
	"github.com/theopenlane/entx"
)

// AnnotationName is the name of the group by field annotation
const AnnotationName = "OPENLANE_GROUP_BY"

var _ schema.Annotation = (*Annotation)(nil)

// Annotation marks an ent field as one a generated group by query can group on
type Annotation struct {
	// Groupable indicates the field can be grouped on
	Groupable bool `json:"groupable,omitempty"`
}

// Name returns the name of the annotation
func (a Annotation) Name() string {
	return AnnotationName
}

// Groupable returns an annotation that adds the field to the schema's group by query
func Groupable() *Annotation {
	return &Annotation{Groupable: true}
}

// Decode unmarshals the annotation from the loaded graph representation
func (a *Annotation) Decode(annotation any) error {
	return entx.DecodeAnnotation(annotation, a)
}
