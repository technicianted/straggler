package types

import pacertypes "stagger/pkg/pacer/types"

type StaggerGroup struct {
	// group name. must be unique.
	Name string
	// set of labels to apply this staggering configuration.
	LabelSelector map[string]string
	// jsonpath aggregation grouping expression.
	GroupingExpression string

	PacerFactory pacertypes.PacerFactory
}
