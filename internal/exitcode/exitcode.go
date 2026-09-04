// Package exitcode defines the shared QYVORA process exit-code contract.
//
//	0   success
//	1   runtime failure (analysis error, I/O error, internal error)
//	2   usage error (unknown flag/command, invalid value, missing/invalid
//	       target, illegal authorization state)
//	130 interrupted (128 + SIGINT)
//
// sekhmet additionally reserves exit code 3 for "unsupported" conditions — a
// request the tool understood but honestly cannot serve (e.g. no execution
// backend for a target type). This mirrors aksum's extension so orchestrators
// can skip rather than retry.
//
// Automation must distinguish these without parsing human output. Usage
// errors are never reported as 1; interrupts always flush any open event
// stream and close connections before exiting.
package exitcode

const (
	Success     = 0
	Runtime     = 1
	Usage       = 2
	Unsupported = 3
	Interrupted = 130
)
