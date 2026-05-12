package utils

import (
	"strconv"
	"strings"
	"time"
)

func TimeFromID(id string) string {
	parts := strings.SplitN(id, "-", 2)
	if len(parts) == 0 {
		return "-"
	}

	ms, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return "-"
	}

	return time.UnixMilli(ms).Format("2006-01-02 15:04:05.000 -07:00 MST")
}