package mc

import (
	"fmt"
	"regexp"
)

func ParseVersion(resp string) string {
	// This server is running Paper version 1.21.10-130-ver/1.21.10@8043efd (2026-01-04T21:00:59Z) (Implementing API version 1.21.10-R0.1-SNAPSHOT)
	re := regexp.MustCompile(`running\s+(\w+)\s+version\s+([\d\.]+)`)
	matches := re.FindStringSubmatch(resp)

	if len(matches) >= 3 {
		serverType := matches[1]
		version := matches[2]
		
		return fmt.Sprintf("%s %s", serverType, version)
	} else {
		return ""
	}
}
