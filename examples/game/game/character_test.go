package game

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestXpForLevel(t *testing.T) {
	tests := []struct {
		level    int
		expected int
	}{
		{level: 1, expected: 0},
		{level: 2, expected: 100},
		{level: 3, expected: 324},
		{level: 4, expected: 647},
		{level: 5, expected: 1055},
	}

	for _, test := range tests {
		result := xpForLevel(test.level)
		require.Equal(t, test.expected, result)
	}
}

func TestGainExperience(t *testing.T) {
	char := Character{
		ID:         uuid.New(),
		Name:       "Test Character",
		Level:      1,
		Experience: 0,
		Status:     StatusIdle,
		Cooldown:   0,
		Action:     nil,
	}

	char.GainExperience(150)
	require.Equal(t, 150, char.Experience)
	require.Equal(t, 2, char.Level)

	char.GainExperience(200)
	require.Equal(t, 350, char.Experience)
	require.Equal(t, 3, char.Level)

	char.GainExperience(500)
	require.Equal(t, 850, char.Experience)
	require.Equal(t, 4, char.Level)
}
