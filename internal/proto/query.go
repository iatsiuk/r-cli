package proto

// QueryType identifies the type of query sent to the server.
type QueryType int

const (
	QueryStart       QueryType = 1
	QueryContinue    QueryType = 2
	QueryStop        QueryType = 3
	QueryNoreplyWait QueryType = 4
	QueryServerInfo  QueryType = 5
)
