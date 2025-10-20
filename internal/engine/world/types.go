// Package world implements world state and entity models.
package world

// ID is a stable identifier for entities, e.g., "SYS:SOL", "FLEET:1234".
type ID string

// Snapshot provides read-only access to world state.
type Snapshot interface {
	GetEntity(id ID) (Entity, bool)
	// TODO: Add indexes and queries
}

// Mutable provides write access to world state.
type Mutable interface {
	Snapshot
	Upsert(Entity)
	Delete(ID)
	// Internal, used by commit stage
}

// Entity represents a world entity.
type Entity interface {
	ID() ID
	Kind() string
	// Serialize/deserialize methods as needed
	String() string
}
