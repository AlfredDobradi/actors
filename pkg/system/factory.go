package system

func WithPreStartHook(hook Hook) HandlerOpt {
	return func(h *ActorHandler, _ *System) {
		if h.preStartHooks == nil {
			h.preStartHooks = NewHookCollection(HookPreStart)
		}
		h.preStartHooks.Add(hook)
	}
}

func WithPostStartHook(hook Hook) HandlerOpt {
	return func(h *ActorHandler, _ *System) {
		if h.postStartHooks == nil {
			h.postStartHooks = NewHookCollection(HookPostStart)
		}
		h.postStartHooks.Add(hook)
	}
}

func WithPoisonedHook(hook Hook) HandlerOpt {
	return func(h *ActorHandler, _ *System) {
		if h.poisonedHooks == nil {
			h.poisonedHooks = NewHookCollection(HookPoisoned)
		}
		h.poisonedHooks.Add(hook)
	}
}

func WithTerminatedHook(hook Hook) HandlerOpt {
	return func(h *ActorHandler, _ *System) {
		if h.terminatedHooks == nil {
			h.terminatedHooks = NewHookCollection(HookTerminated)
		}
		h.terminatedHooks.Add(hook)
	}
}

func WithCrashHook(hook Hook) HandlerOpt {
	return func(h *ActorHandler, _ *System) {
		if h.crashHooks == nil {
			h.crashHooks = NewHookCollection(HookCrash)
		}
		h.crashHooks.Add(hook)
	}
}

// func WithSubscription(pattern string) HandlerOpt {
// 	return func(h *ActorHandler, sys *System) {
// 		registry := sys.registry
// 		registry.Subscribe(pattern, h.actor.GetID())
// 	}
// }
