package proto

// Version identifies the RethinkDB handshake protocol version.
// Sent as a 4-byte little-endian magic number at connection start.
type Version uint32

const (
	// V1_0 uses SCRAM-SHA-256 authentication (current).
	V1_0 Version = 0x34c2bdc3
	// V0_4 is a legacy protocol version.
	V0_4 Version = 0x400c2d20
	// V0_3 is a legacy protocol version.
	V0_3 Version = 0x5f75e83e
	// V0_2 is a legacy protocol version.
	V0_2 Version = 0x723081e1
	// V0_1 is the initial protocol version.
	V0_1 Version = 0x3f61ba36
)
