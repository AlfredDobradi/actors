package paseto

import (
	"bytes"
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
	slog.SetDefault(slog.New(slog.NewTextHandler(writer, nil)))

	os.Unsetenv("PASETO_KEY")
	key := GetKey()
	require.NotNil(t, key)
	require.Contains(t, writer.String(), "PASETO_KEY environment variable is not set or invalid, falling back to random key")
}
