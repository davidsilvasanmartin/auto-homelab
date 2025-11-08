package system

import (
	"testing"
	"time"
)

func TestDefaultTime_Sleep_CallsStdlibSleep(t *testing.T) {
	var capturedDuration time.Duration
	std := &mockStdlib{
		sleep: func(d time.Duration) {
			capturedDuration = d
		},
	}
	dt := &DefaultTime{stdlib: std}
	expectedDuration := 7 * time.Second

	dt.Sleep(expectedDuration)

	if capturedDuration != expectedDuration {
		t.Errorf("expected sleep to have been called with duration %v, got: %v", expectedDuration, capturedDuration)
	}
}
