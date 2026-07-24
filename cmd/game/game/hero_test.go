package game

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHeroPriceMultiplier(t *testing.T) {
	type testCase struct {
		heroAmount int
		expected   int64
	}

	testCases := []testCase{
		{heroAmount: 0, expected: 1},
		{heroAmount: 1, expected: 1},
		{heroAmount: 3, expected: 1},
		{heroAmount: 4, expected: 2},
		{heroAmount: 6, expected: 2},
		{heroAmount: 7, expected: 3},
		{heroAmount: 12, expected: 3},
		{heroAmount: 13, expected: 4},
		{heroAmount: 21, expected: 4},
		{heroAmount: 22, expected: 5},
	}

	for _, tc := range testCases {
		tf := func(t *testing.T) {
			result := HeroPriceMultiplier(tc.heroAmount)
			require.Equal(t, tc.expected, result)
		}

		t.Run(fmt.Sprintf("heroAmount=%d", tc.heroAmount), tf)
	}
}

func TestHeroDecideWhatToDo(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	tests := []struct {
		label                 string
		health                int
		energy                int
		gold                  int
		expectedEitherActions []string
	}{
		{health: 30, energy: 100, gold: 1000, expectedEitherActions: []string{ActionNameHeal}},
		{health: 100, energy: 30, gold: 1000, expectedEitherActions: []string{ActionNameRest}},
		{health: 30, energy: 20, gold: 1000, expectedEitherActions: []string{ActionNameHeal}},
		{health: 100, energy: 100, gold: 1000, expectedEitherActions: []string{ActionNameTavern, ActionNameAdventure, ActionNameGather, ActionNameIdle}},
	}

	for _, tt := range tests {
		tf := func(t *testing.T) {
			hero := &Hero{
				ID:     uuid.New(),
				Health: tt.health,
				Energy: tt.energy,
				Gold:   tt.gold,
			}

			action := hero.WhatNext()
			require.Contains(t, tt.expectedEitherActions, action.GetName())
		}

		t.Run(tt.label, tf)
	}

}

type TestAction struct {
	name     string
	cooldown int
}

func (a *TestAction) GetName() string {
	return a.name
}

func (a *TestAction) GetCooldown() int {
	return a.cooldown
}

func (a *TestAction) Execute(ctx context.Context, hero *Hero) {
	hero.Experience += 1
}

func (a *TestAction) String() string {
	return fmt.Sprintf("%s-ing", a.name)
}

func TestHeroReplayTicks(t *testing.T) {
	// slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	action := &TestAction{name: "test", cooldown: 15}

	tests := []struct {
		label              string
		ticks              int
		action             Action
		cooldownCheck      func(i int) bool
		expectedExperience int
	}{
		{
			label:  "Replay 10 ticks on a 15 tick action",
			ticks:  -10,
			action: action,
			cooldownCheck: func(i int) bool {
				return i == 5
			},
			expectedExperience: 0,
		},
		{
			label:  "Replay 20 ticks on a 15 tick action",
			ticks:  -20,
			action: action,
			cooldownCheck: func(i int) bool {
				return i > 0
			},
			expectedExperience: 1,
		},
	}

	for _, tt := range tests {
		tf := func(t *testing.T) {
			hero := &Hero{
				ID:       uuid.New(),
				Health:   100,
				Energy:   100,
				Gold:     1000,
				LastTick: time.Now().Add(time.Duration(tt.ticks) * time.Second),
				Action:   tt.action,
				Cooldown: tt.action.GetCooldown(),
			}

			err := hero.ReplayTicks(context.Background())
			require.NoError(t, err)
			require.True(t, tt.cooldownCheck(hero.Cooldown), "Cooldown after replaying ticks did not match expected value")
			require.Equal(t, tt.expectedExperience, hero.Experience, "Experience after replaying ticks did not match expected value")
		}

		t.Run(tt.label, tf)
	}
}

func BenchmarkHeroReplayTicksIncremental(b *testing.B) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})))
	hero := &Hero{
		ID:        uuid.New(),
		Health:    100,
		Energy:    100,
		Gold:      1000,
		Action:    nil,
		Inventory: NewInventory(),
	}

	ticks := 20

	for b.Loop() {
		for range ticks {
			hero.ProcessTick(context.Background())
		}
	}
}

func BenchmarkHeroReplayTicksOptimized(b *testing.B) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})))
	hero := &Hero{
		ID:        uuid.New(),
		Health:    100,
		Energy:    100,
		Gold:      1000,
		LastTick:  time.Now().Add(-20 * time.Second),
		Action:    nil,
		Inventory: NewInventory(),
	}

	for b.Loop() {
		hero.ReplayTicks(context.Background())
	}
}
