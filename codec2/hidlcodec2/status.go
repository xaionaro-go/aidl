package hidlcodec2

import "fmt"

// Status is the HIDL Codec2 status code.
type Status int32

const (
	StatusOK        Status = 0
	StatusBadValue  Status = -22
	StatusBadIndex  Status = -75
	StatusCannotDo  Status = -2147483646
	StatusDuplicate Status = -17
	StatusNotFound  Status = -2
	StatusBadState  Status = -38
	StatusBlocking  Status = -9930
	StatusNoMemory  Status = -12
	StatusRefused   Status = -1
	StatusTimedOut  Status = -110
	StatusOmitted   Status = -74
	StatusCorrupted Status = -2147483648
	StatusNoInit    Status = -19
)

// Err returns nil if status is OK, otherwise an error.
func (s Status) Err() error {
	if s == StatusOK {
		return nil
	}
	return fmt.Errorf("codec2 status %d (%s)", int32(s), s.String())
}

// String returns a human-readable name for the status code.
func (s Status) String() string {
	switch s {
	case StatusOK:
		return "OK"
	case StatusBadValue:
		return "BAD_VALUE"
	case StatusBadIndex:
		return "BAD_INDEX"
	case StatusCannotDo:
		return "CANNOT_DO"
	case StatusDuplicate:
		return "DUPLICATE"
	case StatusNotFound:
		return "NOT_FOUND"
	case StatusBadState:
		return "BAD_STATE"
	case StatusBlocking:
		return "BLOCKING"
	case StatusNoMemory:
		return "NO_MEMORY"
	case StatusRefused:
		return "REFUSED"
	case StatusTimedOut:
		return "TIMED_OUT"
	case StatusOmitted:
		return "OMITTED"
	case StatusCorrupted:
		return "CORRUPTED"
	case StatusNoInit:
		return "NO_INIT"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int32(s))
	}
}
