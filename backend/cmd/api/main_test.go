package main

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGinRunsInReleaseModeOnlyInProduction(t *testing.T) {
	require.Equal(t, gin.ReleaseMode, ginMode("production"))
	require.Equal(t, gin.DebugMode, ginMode("development"))
	require.Equal(t, gin.DebugMode, ginMode("test"))
}
