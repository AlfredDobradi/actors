package system

import "log/slog"

type HookOpt func(h *ActorHandler, sys *System)

func WithPreStartHook(hook Hook) HookOpt {
	return func(h *ActorHandler, _ *System) {
		if h.preStartHooks == nil {
			h.preStartHooks = NewHookCollection(HookPreStart)
		}
		h.preStartHooks.Add(hook)
	}
}

func WithPostStartHook(hook Hook) HookOpt {
	return func(h *ActorHandler, _ *System) {
		if h.postStartHooks == nil {
			h.postStartHooks = NewHookCollection(HookPostStart)
		}
		h.postStartHooks.Add(hook)
	}
}

func WithPoisonedHook(hook Hook) HookOpt {
	return func(h *ActorHandler, _ *System) {
		if h.poisonedHooks == nil {
			h.poisonedHooks = NewHookCollection(HookPoisoned)
		}
		h.poisonedHooks.Add(hook)
	}
}

func WithTerminatedHook(hook Hook) HookOpt {
	return func(h *ActorHandler, _ *System) {
		if h.terminatedHooks == nil {
			h.terminatedHooks = NewHookCollection(HookTerminated)
		}
		h.terminatedHooks.Add(hook)
	}
}

func WithCrashHook(hook Hook) HookOpt {
	return func(h *ActorHandler, _ *System) {
		if h.crashHooks == nil {
			h.crashHooks = NewHookCollection(HookCrash)
		}
		h.crashHooks.Add(hook)
	}
}

func WithSubscription(pattern string) HandlerOpt {
	return func(h *ActorHandler, sys *System) {
		if err := sys.Subscribe(pattern, h.actor.GetID()); err != nil {
			slog.Warn("Failed to subscribe actor to pattern", "pattern", pattern, "actorID", h.actor.GetID(), "error", err)
		}
	}
}
