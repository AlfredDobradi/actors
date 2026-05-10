package paseto

import (
	"bytes"
	"encoding/hex"
	"log/slog"
	"os"
	"testing"

	"aidanwoods.dev/go-paseto"
	"github.com/stretchr/testify/require"
)

func TestFromEnv(t *testing.T) {
	// Set a known key in the environment for testing
	testKey := paseto.NewV4SymmetricKey()

	hex := testKey.ExportHex()
	os.Setenv("PASETO_KEY", hex)
	defer os.Unsetenv("PASETO_KEY")

	key := GetKey()
	require.Equal(t, hex, key.ExportHex())
}

func TestFallback(t *testing.T) {
	writer := bytes.NewBufferString("")
	slog.SetDefault(slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	os.Unsetenv("PASETO_KEY")
	symmetricKey = nil
	key := GetKey()
	actual := writer.String()
	require.NotNil(t, key)
	require.Contains(t, actual, "PASETO_KEY environment variable is not set or invalid, falling back to random key")
}

func TestReadKeyFromEnv(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		expectError error
	}{
		{"ValidKey", paseto.NewV4SymmetricKey().ExportHex(), nil},
		{"EmptyKey", "", errEmptyKey},
		{"InvalidHex", "invalidhexstring", hex.InvalidByteError(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("PASETO_KEY", tt.envValue)
			defer os.Unsetenv("PASETO_KEY")

			key, err := readKeyFromEnv()
			if tt.expectError != nil {
				require.Error(t, err)
				require.IsType(t, tt.expectError, err)
				require.Nil(t, key)
			} else {
				require.NoError(t, err)
				require.NotNil(t, key)
				require.Equal(t, tt.envValue, key.ExportHex())
			}
		})
	}
}
