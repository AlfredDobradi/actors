package game

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

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
