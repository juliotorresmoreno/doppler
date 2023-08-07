package helper

import "strconv"

func ParseIntParams(val string, defaultValue int) int {
	result := defaultValue
	if val != "" {
		tlimit, _ := strconv.Atoi(val)
		if tlimit > 0 {
			result = tlimit
		}
	}
	return result
}
