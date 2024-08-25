package obsgrpcproxy

import "context"

type EventHook interface {
	ProcessEvent(ctx context.Context, event any)
}

type configT struct {
	EventHooks []EventHook
}

type Option interface {
	apply(cfg *configT)
}

type Options []Option

func (s Options) apply(cfg *configT) {
	for _, opt := range s {
		opt.apply(cfg)
	}
}

func (s Options) config() configT {
	cfg := configT{}
	s.apply(&cfg)
	return cfg
}

type OptionEventHook struct{ EventHook }

func (opt OptionEventHook) apply(cfg *configT) {
	cfg.EventHooks = append(cfg.EventHooks, opt.EventHook)
}
