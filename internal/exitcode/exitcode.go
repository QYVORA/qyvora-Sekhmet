// Package exitcode defines the shared QYVORA process exit-code contract:
//
//	0   success
//	1   runtime failure (analysis error, I/O error, internal error)
//	2   usage error (unknown flag/command, invalid value, missing/invalid
//	       target flag combination)
//	3   authorization refused (command requires an authorized target but the
//	       current target is not authorized) — distinct from usage so automation
//	       can distinguish "bad invocation" from "recognised but declined"
//	130 interrupted (128 + SIGINT)
//
// Automation must distinguish these without parsing human output. Usage
// errors are never reported as 1; interrupts always flush any open event
// stream and close connections before exiting.
package exitcode

const (
	Success              = 0
	Runtime              = 1
	Usage                = 2
	AuthorizationRefused = 3
	Interrupted          = 130
)
