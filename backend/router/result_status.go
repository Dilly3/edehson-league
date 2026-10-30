package router

import "fmt"

func requestedResultStatus(requested string) (string, error) {
	// Preserve compatibility with older clients that publish final results.
	if requested == "" {
		return "finalized", nil
	}
	switch requested {
	case "live", "half_time", "finalized":
		return requested, nil
	default:
		return "", fmt.Errorf("status must be live, half_time or finalized")
	}
}

func canPublishStatus(current, next string) bool {
	if next != "live" && next != "half_time" && next != "finalized" {
		return false
	}
	switch current {
	case "scheduled", "live", "half_time":
		return true
	case "finalized":
		return next == "finalized"
	default:
		return false
	}
}
