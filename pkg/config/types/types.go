// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package types

import (
	"time"

	pacertypes "straggler/pkg/pacer/types"
)

type StaggerGroup struct {
	// group name. must be unique.
	Name string
	// set of labels to apply this staggering configuration.
	LabelSelector map[string]string
	// set of labels to bypass staggering.
	BypassLabelSelector map[string]string
	// jsonpath aggregation grouping expression.
	GroupingExpression string
	// Maximum time to keep a pod in blocked state. Default none.
	MaxBlockedDuration time.Duration

	PacerFactory pacertypes.PacerFactory
}
