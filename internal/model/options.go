package model

import "time"

type Options struct {
	RestartThreshold int
	PendingThreshold time.Duration
}
