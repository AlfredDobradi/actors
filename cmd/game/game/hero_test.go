package game

import (
	"fmt"
	"testing"

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
