package client

const (
	ExampleSpecType            = "exampleSpec"
	ExampleSpecFieldExampleTwo = "exampleOne"
)

type ExampleSpec struct {
	ExampleTwo int64 `json:"exampleOne,omitempty" yaml:"exampleOne,omitempty"`
}
