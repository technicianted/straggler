// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package linear

import "straggler/pkg/pacer/types"

var _ types.PacerFactory = &factory{}

type factory struct {
	config Config
}

func NewFactory(config Config) *factory {
	return &factory{
		config: config,
	}
}

func (f *factory) New(key string) types.Pacer {
	return New(key, f.config)
}
