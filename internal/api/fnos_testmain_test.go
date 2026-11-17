package api

import (
	"context"
	"os"
	"testing"

	"github.com/fonu/fonu/internal/fnos"
)

func TestMain(m *testing.M) {
	fnos.Find = func(context.Context, string) (string, bool) { return "", false }
	os.Exit(m.Run())
}
