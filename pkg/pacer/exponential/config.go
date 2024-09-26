// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package exponential

type Config struct {
	// Minimum number of pods to initially allow.
	MinInitial int
	// Maximum number of staggered pods after which it's disabled.
	MaxStagger int
	// Exponential staggering multiplier
	Multiplier float64
}
