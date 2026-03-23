package engine

// STS operations are identified by the Action form parameter, not by path.
// All STS requests go to POST / with Action=<operation>.

var stsActions = map[string]string{
	"AssumeRole":        "AssumeRole",
	"GetCallerIdentity": "GetCallerIdentity",
}
