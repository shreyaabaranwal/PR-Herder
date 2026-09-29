
package scheduler

import "time"


func IsQuietHours(t time.Time, startHour, endHour int) bool {
	if startHour == endHour {
		return false
	}
	hour := t.Hour()
	if startHour < endHour {
		return hour >= startHour && hour < endHour
	}
	
	return hour >= startHour || hour < endHour
}