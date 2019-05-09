package client

const (
	ExampleConditionType                    = "exampleCondition"
	ExampleConditionFieldLastTransitionTime = "lastTransitionTime"
	ExampleConditionFieldLastUpdateTime     = "lastUpdateTime"
	ExampleConditionFieldMessage            = "message"
	ExampleConditionFieldReason             = "reason"
	ExampleConditionFieldStatus             = "status"
	ExampleConditionFieldType               = "type"
)

type ExampleCondition struct {
	LastTransitionTime string `json:"lastTransitionTime,omitempty" yaml:"lastTransitionTime,omitempty"`
	LastUpdateTime     string `json:"lastUpdateTime,omitempty" yaml:"lastUpdateTime,omitempty"`
	Message            string `json:"message,omitempty" yaml:"message,omitempty"`
	Reason             string `json:"reason,omitempty" yaml:"reason,omitempty"`
	Status             string `json:"status,omitempty" yaml:"status,omitempty"`
	Type               string `json:"type,omitempty" yaml:"type,omitempty"`
}
