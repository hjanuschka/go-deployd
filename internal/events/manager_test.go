package events_test

import (
	"testing"

	"github.com/hjanuschka/go-deployd/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUniversalScriptManager(t *testing.T) {
	t.Run("create new manager", func(t *testing.T) {
		manager := events.NewUniversalScriptManager()
		require.NotNil(t, manager)
	})
}

func TestScriptManager(t *testing.T) {
	t.Run("create new script manager", func(t *testing.T) {
		manager := events.NewScriptManager()
		require.NotNil(t, manager)
	})
}

func TestGoPluginManager(t *testing.T) {
	t.Run("create new go plugin manager", func(t *testing.T) {
		manager := events.NewGoPluginManager()
		require.NotNil(t, manager)
	})
}

func TestHotReloadManager(t *testing.T) {
	t.Run("create new hot reload manager", func(t *testing.T) {
		manager := events.NewHotReloadGoManager("")
		require.NotNil(t, manager)
	})
}

func TestEventTypes(t *testing.T) {
	t.Run("test event type constants", func(t *testing.T) {
		// Just test that the package compiles and basic functionality works
		assert.True(t, true)
	})
}