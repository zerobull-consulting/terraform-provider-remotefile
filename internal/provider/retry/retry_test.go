package retry

import (
	"errors"
	"testing"
	"time"
)

func TestWithRetrySuccessFirstTry(t *testing.T) {
	err := WithRetry(3, time.Millisecond, func() error {
		return nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWithRetrySuccessAfterRetries(t *testing.T) {
	startTime := time.Now()
	count := 0
	err := WithRetry(3, time.Millisecond, func() error {
		if count < 2 {
			count++
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	elapsed := time.Since(startTime)
	minExpectedDuration := 2 * time.Millisecond
	if elapsed < minExpectedDuration {
		t.Errorf("operation completed too quickly: %v < %v", elapsed, minExpectedDuration)
	}
}

func TestWithRetryFailureAfterAllRetries(t *testing.T) {
	startTime := time.Now()
	expectedErr := errors.New("persistent error")
	err := WithRetry(2, time.Millisecond, func() error {
		return expectedErr
	})

	if err == nil {
		t.Error("expected error but got none")
	}
	if err.Error() != expectedErr.Error() {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	elapsed := time.Since(startTime)
	minExpectedDuration := 2 * time.Millisecond
	if elapsed < minExpectedDuration {
		t.Errorf("operation completed too quickly: %v < %v", elapsed, minExpectedDuration)
	}
}

func TestWithRetryZeroRetries(t *testing.T) {
	expectedErr := errors.New("immediate error")
	err := WithRetry(0, time.Millisecond, func() error {
		return expectedErr
	})

	if err == nil {
		t.Error("expected error but got none")
	}
	if err.Error() != expectedErr.Error() {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}
