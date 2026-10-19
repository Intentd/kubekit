package types

// Result says what a write did. Every verb is idempotent: each result below
// comes with a nil error, so a caller that only cares about failures can
// ignore it.
type Result string

const (
	Created   Result = "created"   // Create: it did not exist.
	Unchanged Result = "unchanged" // It already existed, or already matched; left alone.
	Applied   Result = "applied"   // Apply: written with server-side apply.
	Replaced  Result = "replaced"  // Apply: an immutable field changed, so it was deleted and created again.
	Deleted   Result = "deleted"   // Delete, DeleteIfOwned.
	NotFound  Result = "not found" // Delete, DeleteIfOwned: it was already gone.
	NotOwned  Result = "not owned" // DeleteIfOwned: someone else owns it; left alone.
)
