// Package schedule implements dependency graphs and batching.
package schedule

import (
	"github.com/playbymail/fh/internal/engine/orders"
)

// Batch groups orders for parallel execution.
type Batch struct {
	Orders []orders.Order
}

// Planner builds execution batches from order dependencies.
type Planner interface {
	Plan([]orders.Order, ReadOnlyWorld) ([]Batch, error)
}

// ReadOnlyWorld is a world view for planning.
type ReadOnlyWorld interface {
	GetEntity(id string) (interface{}, bool)
}
