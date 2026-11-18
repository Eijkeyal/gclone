// Package adapters is the list of ecosystems gclone knows. Adding an ecosystem means
// adding it here.
package adapters

import (
	"github.com/rauni5/gclone/internal/adapter"
	"github.com/rauni5/gclone/internal/adapter/flutter"
	"github.com/rauni5/gclone/internal/adapter/golang"
	"github.com/rauni5/gclone/internal/adapter/java"
	"github.com/rauni5/gclone/internal/adapter/node"
	"github.com/rauni5/gclone/internal/adapter/python"
)

// All returns the adapters in the order they are analyzed and shown.
func All() []adapter.Adapter {
	return []adapter.Adapter{node.Adapter{}, python.Adapter{}, golang.Adapter{}, java.Adapter{}, flutter.Adapter{}}
}
