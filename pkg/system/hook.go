package system

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	HookPreStart   = "preStart"
	HookPostStart  = "postStart"
	HookPoisoned   = "poisoned"
	HookTerminated = "terminated"
	HookCrash      = "crash"
)

type Hook func(context.Context, Actor) error

type HookCollection struct {
	name  string
	hooks []Hook
}

func (hc *HookCollection) Add(hook Hook) {
	hc.hooks = append(hc.hooks, hook)
}

func NewHookCollection(name string, hooks ...Hook) *HookCollection {
	return &HookCollection{
		name:  name,
		hooks: hooks,
	}
}

func (hc *HookCollection) run(_ context.Context, actor Actor) {
	subContext, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	wg := &sync.WaitGroup{}
	for _, h := range hc.hooks {
		wg.Add(1)
		go func(hook Hook) {
			defer wg.Done()
			if err := hook(subContext, actor); err != nil {
				slog.Warn("Error running hook", "hook", hc.name, "error", err)
			}
		}(h)
	}
	wg.Wait()

	if err := subContext.Err(); err != nil {
		slog.Warn("Hook execution context error", "hook", hc.name, "error", err)
		return
	}

	slog.Debug("Completed running hooks", "hook", hc.name, "count", len(hc.hooks))
}
