package proto

// ResponseType identifies the type of response from the server.
type ResponseType int

const (
	ResponseSuccessAtom     ResponseType = 1
	ResponseSuccessSequence ResponseType = 2
	ResponseSuccessPartial  ResponseType = 3
	ResponseWaitComplete    ResponseType = 4
	ResponseServerInfo      ResponseType = 5
	ResponseClientError     ResponseType = 16
	ResponseCompileError    ResponseType = 17
	ResponseRuntimeError    ResponseType = 18
)

// IsError reports whether the response type represents an error condition.
func (r ResponseType) IsError() bool {
	return r >= 16
}

// ErrorType identifies the kind of runtime error from the server.
type ErrorType int

const (
	ErrorInternal        ErrorType = 1000000
	ErrorResourceLimit   ErrorType = 2000000
	ErrorQueryLogic      ErrorType = 3000000
	ErrorNonExistence    ErrorType = 3100000
	ErrorOpFailed        ErrorType = 4100000
	ErrorOpIndeterminate ErrorType = 4200000
	ErrorUser            ErrorType = 5000000
	ErrorPermission      ErrorType = 6000000
)

// ResponseNote carries metadata about cursor/feed type in a response.
type ResponseNote int

const (
	NoteSequenceFeed     ResponseNote = 1
	NoteAtomFeed         ResponseNote = 2
	NoteOrderByLimitFeed ResponseNote = 3
	NoteUnionedFeed      ResponseNote = 4
	NoteIncludesStates   ResponseNote = 5
)
