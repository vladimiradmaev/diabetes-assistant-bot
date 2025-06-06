package utils

import "time"

// TimeToMinutes converts time string to minutes since midnight
func TimeToMinutes(timeStr string) int {
	t, _ := time.Parse("15:04", timeStr)
	return t.Hour()*60 + t.Minute()
}
