package helper

import "strconv"

func ParseIntParams(val string, defaultValue int) int {
	result := defaultValue
	if val != "" {
		value, _ := strconv.Atoi(val)
		if value > 0 {
			result = value
		}
	}
	return result
}
