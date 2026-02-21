package proto

// MaxFrameSize is the maximum allowed payload size per the RethinkDB wire protocol (64MB).
const MaxFrameSize uint32 = 64 * 1024 * 1024
