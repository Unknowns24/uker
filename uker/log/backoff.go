package log

import (
	"time"

	"golang.org/x/exp/rand"
)

func backoff(attempt int) time.Duration {
	min := 1 * time.Second
	max := 1 * time.Hour

	sleep := min * time.Duration(1<<attempt)
	if sleep > max {
		sleep = max
	}

	jitter := time.Duration(rand.Int63n(int64(min)))
	return sleep + jitter
}

// RetryWithBackoff keeps retrying the provided operation using an exponential backoff.
func RetryWithBackoff(operation func() error) {
	for attempt := 0; ; attempt++ {
		if err := operation(); err == nil {
			return
		}

		time.Sleep(backoff(attempt))
	}
}
