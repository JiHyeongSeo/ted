package types

// GutterMark represents git diff status for a line in the gutter.
type GutterMark int

const (
	MarkNone     GutterMark = iota
	MarkAdded
	MarkModified
	MarkDeleted
)
