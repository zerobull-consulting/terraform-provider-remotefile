package retry

import "time"

func WithRetry(retryCount int64, retryInterval time.Duration, operation func() error) error {
	var lastErr error

	for i := int64(0); i <= retryCount; i++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		if i == retryCount {
			return lastErr
		}

		time.Sleep(retryInterval)
	}

	return lastErr
}
