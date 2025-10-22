// Package effects implements write-set buffers and merge semantics.
package effects

// Buffer collects effects from order execution.
type Buffer interface {
	Add(effect interface{})
	Merge()                  // Merge conflicting effects
	Apply(world interface{}) // Apply to mutable world
}
