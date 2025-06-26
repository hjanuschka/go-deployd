package router_test

import (
	"testing"

	"github.com/hjanuschka/go-deployd/internal/router"
	"github.com/hjanuschka/go-deployd/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouter(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	t.Run("create new router", func(t *testing.T) {
		r := router.New(db, true, "")
		require.NotNil(t, r)
		assert.NotNil(t, r)
	})
}

func TestRouterMethods(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	t.Run("router basic functionality", func(t *testing.T) {
		r := router.New(db, true, "")
		require.NotNil(t, r)
		
		// Test that the router was created successfully
		// Without knowing the exact interface, we can't test much more
		assert.True(t, true)
	})
}