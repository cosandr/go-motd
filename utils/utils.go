package utils

import "fmt"

// Wrap `s` around start and end, returns <start>s<end>
func Wrap(s string, start string, end string) string {
	return fmt.Sprintf("%s%s%s", start, s, end)
}
