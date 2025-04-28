package elephas

// https://www.postgresql.org/docs/current/protocol-message-formats.html

// type MessageId byte

const (
	errorResponseMsg = 'E'
	parameterStatus  = 'S'
	authMsgType      = 'R'
	backendKeyData   = 'K'
	readyForQuery    = 'Z'
	rowDescription   = 'T'
	dataRow          = 'D'
	commandComplete  = 'C'
	queryCommand     = 'Q'
	parseCommand     = 'P'
	bindCommand      = 'B'
	executeCommand   = 'E'
	flushCommand     = 'H'
	syncCommand      = 'S'
	describeCommand  = 'D'
	parseComplete    = '1'

	SASL         = 10
	SASLContinue = 11
	SASLComplete = 12
	AuthSuccess  = 0
)

type TransactionStatus int

// https://www.postgresql.org/docs/current/protocol-message-formats.html#PROTOCOL-MESSAGE-FORMATS-READYFORQUERY
const (
	I TransactionStatus = 73
	T TransactionStatus = 84
	E TransactionStatus = 69
)

type CommandTag string

// https://www.postgresql.org/docs/current/protocol-message-formats.html#PROTOCOL-MESSAGE-FORMATS-COMMANDCOMPLETE
const (
	beginCmd    CommandTag = "BEGIN"
	rollbackCmd CommandTag = "ROLLBACK"
	commitCmd   CommandTag = "COMMIT"
)

type Format uint16

const (
	fmtText Format = iota
	fmtBinary
)
