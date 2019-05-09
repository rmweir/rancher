package client

const (
	ExampleStatusType            = "exampleStatus"
	ExampleStatusFieldConditions = "conditions"
)

type ExampleStatus struct {
	Conditions []ExampleCondition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}
