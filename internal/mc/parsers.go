package mc

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func parseTrailingInt(input string) (int, error) {
	input = strings.TrimSpace(input)

	re := regexp.MustCompile(`(\d+)$`)
	m := re.FindStringSubmatch(input)
	if len(m) != 2 {
		return 0, fmt.Errorf("no trailing int in %q", input)
	}
	return strconv.Atoi(m[1])
}

func ParseTime(input string) string {
	re := regexp.MustCompile(`The time is (\d+)`)
	match := re.FindStringSubmatch(input)
	
	if len(match) < 2 {
		return "00:00 AM"
	}

	ticks, _ := strconv.Atoi(match[1])
	ticks = ticks % 24000 // Upewniamy się, że mieścimy się w jednej dobie

	// Minecraft 0 ticks = 6:00 AM
	totalHours := (ticks / 1000 + 6) % 24
	minutes := (ticks % 1000) * 60 / 1000

	period := "AM"
	hour12 := totalHours

	if totalHours >= 12 {
		period = "PM"
		if totalHours > 12 {
			hour12 = totalHours - 12
		}
	}
	if hour12 == 0 {
		hour12 = 12
	}

	return fmt.Sprintf("%02d:%02d %s", hour12, minutes, period)
}

func ParseTPS(input string) (t1, t5, t15 float64) {
	// 1. Usuwamy kolory (§a, §f itp.)
	clean := regexp.MustCompile(`§.`).ReplaceAllString(input, "")

	// 2. Szukamy wszystkiego, co jest po dwukropku
	parts := strings.Split(clean, ":")
	if len(parts) < 2 {
		return 0, 0, 0
	}

	// 3. Wyciągamy liczby tylko z części po dwukropku
	re := regexp.MustCompile(`[\d\.]+`)
	matches := re.FindAllString(parts[1], -1)

	if len(matches) < 3 {
		return 0, 0, 0
	}

	// Helper do konwersji
	pf := func(s string) float64 {
		val, _ := strconv.ParseFloat(s, 64)
		return val
	}

	return pf(matches[0]), pf(matches[1]), pf(matches[2])
}

type Vec3 struct {
	X, Y, Z float64
}

func ParsePosition(input string) (Vec3, error) {
	re := regexp.MustCompile(`\[(.+?)d, (.+?)d, (.+?)d\]`)
	m := re.FindStringSubmatch(input)
	if len(m) != 4 {
		return Vec3{}, errors.New("invalid position output")
	}

	x, _ := strconv.ParseFloat(m[1], 64)
	y, _ := strconv.ParseFloat(m[2], 64)
	z, _ := strconv.ParseFloat(m[3], 64)

	return Vec3{x, y, z}, nil
}

func ParseHealth(input string) (float64, error) {
	re := regexp.MustCompile(`: ([0-9.]+)f`)
	m := re.FindStringSubmatch(input)
	if len(m) != 2 {
		return 0, errors.New("invalid health output")
	}
	return strconv.ParseFloat(m[1], 64)
}

func ParseFoodLevel(input string) (int, error) {
	return parseTrailingInt(input)
}

func ParseXPLevel(input string) (int, error) {
	return parseTrailingInt(input)
}

func ParseXPProgress(input string) (float64, error) {
	return ParseHealth(input)
}

func ParseDimension(input string) (string, error) {
	re := regexp.MustCompile(`"minecraft:(.+)"`)
	m := re.FindStringSubmatch(input)
	if len(m) != 2 {
		return "", errors.New("invalid dimension output")
	}
	return m[1], nil
}

type SelectedItem struct {
	ID       string
	Count    int
	Empty    bool
	HasExtra bool // true jeśli components / enchants / custom_name są obecne
}

func ParseSelectedItem(input string) (SelectedItem, error) {
	input = strings.TrimSpace(input)

	if strings.Contains(input, "Found no elements matching SelectedItem") || input == "{}" {
		return SelectedItem{Empty: true}, nil
	}

	reID := regexp.MustCompile(`id:\s*"([^"]+)"`)
	reCount := regexp.MustCompile(`count:\s*(\d+)`)

	idMatch := reID.FindStringSubmatch(input)
	countMatch := reCount.FindStringSubmatch(input)

	if idMatch == nil || countMatch == nil {
		return SelectedItem{}, errors.New("invalid selected item output")
	}

	strconvCount, err := strconv.Atoi(countMatch[1])
	if err != nil {
		return SelectedItem{}, fmt.Errorf("invalid count value: %v", err)
	}

	return SelectedItem{
		ID:       idMatch[1],
		Count:    strconvCount,
		Empty:    false,
		HasExtra: false,
	}, nil
}

func ParseScoreboardInt(input string) (int, error) {
	input = strings.TrimSpace(input)
	re := regexp.MustCompile(`(\d+)$`)
	m := re.FindStringSubmatch(input)
	if len(m) != 2 {
		return 0, errors.New("invalid scoreboard output")
	}
	return strconv.Atoi(m[1])
}
