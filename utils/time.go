package utils

import "time"

// ToIST converts UTC time to Indian Standard Time (IST)
func ToIST(t time.Time) time.Time {
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return t // Fallback to UTC if IST is not available
	}
	return t.In(ist)
}
