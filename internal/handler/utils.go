package handler

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var monthYearRegex = regexp.MustCompile(`^(0[1-9]|1[0-2])-\d{4}$`)

func isInvalidDate(dateStr string) bool {
	if dateStr == "" {
		return false
	}

	if !monthYearRegex.MatchString(dateStr) {
		return true
	}

	_, err := time.Parse("01-2006", dateStr)
	return err != nil
}

func parseID(idStr string) (int64, error) {
	if idStr == "" || strings.ContainsAny(idStr, "/.\\-+") {
		return 0, fmt.Errorf("invalid id format")
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("id must be positive integer")
	}

	return id, nil
}
