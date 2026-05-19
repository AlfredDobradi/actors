package api_test

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/alfreddobradi/actors/examples/game/api"
	"github.com/alfreddobradi/actors/pkg/database/memory"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/stretchr/testify/require"
)

func startSuite() chan struct{} {
	registry := system.NewRegistry()
	db := memory.NewStore()
	sys := system.MustNewSystem(registry, db)

	s := api.NewServer(sys, db)

	stop := make(chan struct{})
	ctx := context.Background()

	go func() {
		s.Start()
		<-stop
		s.Shutdown(ctx)
		sys.Shutdown(ctx)
		db.Close(ctx)
	}()

	return stop
}

func TestApiIndex(t *testing.T) {
	stop := startSuite()
	defer close(stop)

	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	buf := bytes.NewBufferString("")
	_, err = buf.ReadFrom(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "Welcome to the game API", buf.String())
}
