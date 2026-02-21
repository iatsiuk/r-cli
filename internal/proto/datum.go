package proto

// DatumType identifies the type of a datum value in a response.
type DatumType int

const (
	DatumNull   DatumType = 1
	DatumBool   DatumType = 2
	DatumNum    DatumType = 3
	DatumStr    DatumType = 4
	DatumArray  DatumType = 5
	DatumObject DatumType = 6
	DatumJSON   DatumType = 7
)
