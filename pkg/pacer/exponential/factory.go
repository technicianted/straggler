// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package exponential

import "straggler/pkg/pacer/types"

var _ types.PacerFactory = &factory{}

type factory struct {
	name   string
	config Config
}

func NewFactory(name string, config Config) *factory {
	return &factory{
		name:   name,
		config: config,
	}
}

func (f *factory) New(key string) types.Pacer {
	return New(f.name, key, f.config)
}
