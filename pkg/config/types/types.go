// Copyright (c) stagger team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package types

import pacertypes "stagger/pkg/pacer/types"

type StaggerGroup struct {
	// group name. must be unique.
	Name string
	// set of labels to apply this staggering configuration.
	LabelSelector map[string]string
	// set of labels to bypass staggering.
	BypassLabelSelector map[string]string
	// jsonpath aggregation grouping expression.
	GroupingExpression string

	PacerFactory pacertypes.PacerFactory
}
