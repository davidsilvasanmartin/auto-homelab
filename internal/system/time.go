package system

import (
	"time"
)

type Time interface {
	// Sleep pauses the execution of the program for a certain amount of time
	Sleep(d time.Duration)
}

type DefaultTime struct {
	stdlib stdlib
}

func NewDefaultTime() *DefaultTime {
	return &DefaultTime{
		stdlib: newGoStdlib(),
	}
}

func (t *DefaultTime) Sleep(d time.Duration) {
	t.stdlib.Sleep(d)
}
