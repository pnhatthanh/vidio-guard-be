package constants

type ModerationVerdict string

const (
	VerdictSafe       ModerationVerdict = "safe"
	VerdictWarning    ModerationVerdict = "warning"
	VerdictViolation  ModerationVerdict = "violation"
)

func (v ModerationVerdict) String() string { return string(v) }

func (v ModerationVerdict) Violated() bool {
	return v != VerdictSafe
}
