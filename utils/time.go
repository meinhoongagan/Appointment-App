package utils

import "time"

// ToIST converts UTC time to Indian Standard Time (IST)
func ToIST(t time.Time) time.Time {
	type timeResult struct {
		Time time.Time
	}

	resultChan := make(chan timeResult)

	go func() {
		var result timeResult
		ist, err := time.LoadLocation("Asia/Kolkata")
		if err != nil {
			result.Time = t // Fallback to UTC if IST is not available
			resultChan <- result
			return
		}
		result.Time = t.In(ist)
		resultChan <- result
	}()

	result := <-resultChan
	return result.Time
}
