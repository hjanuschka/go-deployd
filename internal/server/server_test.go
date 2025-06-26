package server_test

import (
	"testing"

	"github.com/hjanuschka/go-deployd/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	t.Run("create server config", func(t *testing.T) {
		config := &server.Config{
			Port:         8080,
			DatabaseType: "sqlite",
			DatabaseName: ":memory:",
			Development:  true,
		}

		require.NotNil(t, config)
		assert.Equal(t, 8080, config.Port)
		assert.Equal(t, "sqlite", config.DatabaseType)
		assert.True(t, config.Development)
	})
}
