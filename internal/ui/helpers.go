package ui

import "strings"

func AsciiBar(percent float64, width int, fillChar string, emptyChar string) string {
	if width <= 0 {
		return "[]"
	}

	if percent < 0 {
		percent = 0
	} else if percent > 1 {
		percent = 1
	}

	filled := int(percent * float64(width))

	return "[" +
		strings.Repeat(fillChar, filled) +
		strings.Repeat(emptyChar, width-filled) +
		"]"
}
