package exponential

import "stagger/pkg/pacer/types"

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
