package response

import (
	"strconv"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

func retryAfter(info *errdetails.RetryInfo) string {
	if info == nil || info.RetryDelay == nil {
		return ""
	}
	delay := info.RetryDelay
	if delay.CheckValid() != nil || delay.Seconds < 0 || delay.Nanos < 0 {
		return ""
	}

	// Valid protobuf seconds fit int64, even after rounding up, but may
	// exceed time.Duration. Calculate seconds directly to avoid saturation.
	seconds := delay.Seconds
	if delay.Nanos > 0 {
		seconds++
	}
	return strconv.FormatInt(seconds, 10)
}
