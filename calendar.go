package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Holiday struct {
	Date string `json:"date"`
	Name string `json:"localName"`
}

type AladhanResponse struct {
	Data struct {
		Hijri struct {
			Date  string `json:"date"`
			Day   string `json:"day"`
			Month string `json:"month"`
			Year  string `json:"year"`
		} `json:"hijri"`
	} `json:"data"`
}

func handleCalendar(args []string) {
	if len(args) < 1 {
		fmt.Println("Calendar subcommands: convert, now, holidays, remind")
		fmt.Println("Run 'weathercli help' for details")
		return
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "convert":
		if len(subArgs) != 3 {
			fmt.Println("Usage: weathercli calendar convert <from> <to> <date>")
			fmt.Println("Example: weathercli calendar convert gregorian islamic 2026-06-12")
			fmt.Println("Supported: gregorian, islamic, persian, hebrew, lunar")
			return
		}
		result, err := convertCalendar(subArgs[0], subArgs[1], subArgs[2])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println(result)

	case "now":
		result, err := showAllCalendars()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Print(result)

	case "holidays":
		if len(subArgs) < 1 {
			fmt.Println("Usage: weathercli calendar holidays <country> [year]")
			return
		}
		country := strings.ToUpper(subArgs[0])
		year := ""
		if len(subArgs) > 1 {
			year = subArgs[1]
		}
		holidays, err := getHolidays(country, year)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Public Holidays for %s %s:\n", country, year)
		for _, h := range holidays {
			fmt.Printf("  %s - %s\n", h.Date, h.Name)
		}

	case "remind":
		if len(subArgs) < 2 {
			fmt.Println("Usage: weathercli calendar remind <name> <date>")
			return
		}
		name := subArgs[0]
		date := subArgs[1]
		err := setReminder(name, date)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

	default:
		fmt.Printf("Unknown calendar subcommand: %s\n", sub)
	}
}

func convertCalendar(from, to, date string) (string, error) {
	var t time.Time
	var err error

	from = strings.ToLower(from)
	to = strings.ToLower(to)

	if from == "gregorian" {
		t, err = time.Parse("2006-01-02", date)
		if err != nil {
			return "", fmt.Errorf("invalid date format: use YYYY-MM-DD")
		}
	} else {
		return "", fmt.Errorf("conversion from %s requires gregorian as source", from)
	}

	switch to {
	case "islamic":
		return gregorianToIslamic(t)
	case "persian":
		return gregorianToPersian(t), nil
	case "hebrew":
		return gregorianToHebrew(t)
	case "lunar":
		return gregorianToChineseLunar(t), nil
	default:
		return "", fmt.Errorf("unsupported target: %s", to)
	}
}

func gregorianToIslamic(t time.Time) (string, error) {
	url := fmt.Sprintf("http://api.aladhan.com/v1/gToH?date=%02d-%02d-%d",
		t.Day(), t.Month(), t.Year())
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result AladhanResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse error: %v", err)
	}

	h := result.Data.Hijri
	return fmt.Sprintf("%s %s %s AH", h.Day, h.Month, h.Year), nil
}

func gregorianToPersian(t time.Time) string {
	gYear, gMonth, gDay := t.Year(), int(t.Month()), t.Day()
	days := gregorianDaysFromEpoch(gYear, gMonth, gDay) - gregorianDaysFromEpoch(622, 3, 20)
	if days < 0 {
		return "Before Persian epoch"
	}

	pYear := days / 365
	remaining := days % 365

	pMonths := []int{31, 31, 31, 31, 31, 31, 30, 30, 30, 30, 30, 29}
	if isPersianLeap(pYear + 1) {
		pMonths[11] = 30
	}

	pMonth := 1
	for i, daysInMonth := range pMonths {
		if remaining < daysInMonth {
			pMonth = i + 1
			break
		}
		remaining -= daysInMonth
	}

	return fmt.Sprintf("%d-%02d-%02d (Persian/Solar Hijri)", pYear+1, pMonth, remaining+1)
}

func isPersianLeap(year int) bool {
	return (year*11+14)%33 < 11
}

func gregorianToHebrew(t time.Time) (string, error) {
	url := fmt.Sprintf("https://www.hebcal.com/converter?cfg=json&gy=%d&gm=%d&gd=%d&g2h=1",
		t.Year(), int(t.Month()), t.Day())
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("API error: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return "Hebrew: API unavailable", nil
	}

	if hy, ok := raw["hy"].(float64); ok {
		if hm, ok := raw["hm"].(string); ok {
			if hd, ok := raw["hd"].(float64); ok {
				return fmt.Sprintf("%s %d, %d (Hebrew)", hm, int(hd), int(hy)), nil
			}
		}
	}
	return "Hebrew: conversion available via hebcal.com", nil
}

func gregorianDaysFromEpoch(year, month, day int) int {
	month = (month + 9) % 12
	year = year - month/10
	return 365*year + year/4 - year/100 + year/400 + (month*306+5)/10 + (day - 1)
}

func gregorianToChineseLunar(t time.Time) string {
	lunarInfo := []int{
		0x04bd8, 0x04ae0, 0x0a570, 0x054d5, 0x0d260, 0x0d950, 0x16554, 0x056a0, 0x09ad0, 0x055d2,
		0x04ae0, 0x0a5b6, 0x0a4d0, 0x0d250, 0x1d255, 0x0b540, 0x0d6a0, 0x0ada2, 0x095b0, 0x14977,
		0x04970, 0x0a4b0, 0x0b4b5, 0x06a50, 0x06d40, 0x1ab54, 0x02b60, 0x09570, 0x052f2, 0x04970,
		0x06566, 0x0d4a0, 0x0ea50, 0x06e95, 0x05ad0, 0x02b60, 0x186e3, 0x092e0, 0x1c8d7, 0x0c950,
		0x0d4a0, 0x1d8a6, 0x0b550, 0x056a0, 0x1a5b4, 0x025d0, 0x092d0, 0x0d2b2, 0x0a950, 0x0b557,
		0x06ca0, 0x0b550, 0x15355, 0x04da0, 0x0a5b0, 0x14573, 0x052b0, 0x0a9a8, 0x0e950, 0x06aa0,
		0x0aea6, 0x0ab50, 0x04b60, 0x0aae4, 0x0a570, 0x05260, 0x0f263, 0x0d950, 0x05b57, 0x056a0,
		0x096d0, 0x04dd5, 0x04ad0, 0x0a4d0, 0x0d4d4, 0x0d250, 0x0d558, 0x0b540, 0x0b6a0, 0x195a6,
		0x095b0, 0x049b0, 0x0a974, 0x0a4b0, 0x0b27a, 0x06a50, 0x06d40, 0x0af46, 0x0ab60, 0x09570,
		0x04af5, 0x04970, 0x064b0, 0x074a3, 0x0ea50, 0x06b58, 0x05ac0, 0x0ab60, 0x096d5, 0x092e0,
		0x0c960, 0x0d954, 0x0d4a0, 0x0da50, 0x07552, 0x056a0, 0x0abb7, 0x025d0, 0x092d0, 0x0cab5,
		0x0a950, 0x0b4a0, 0x0baa4, 0x0ad50, 0x055d9, 0x04ba0, 0x0a5b0, 0x15176, 0x052b0, 0x0a930,
		0x07954, 0x06aa0, 0x0ad50, 0x05b52, 0x04b60, 0x0a6e6, 0x0a4e0, 0x0d260, 0x0ea65, 0x0d530,
		0x05aa0, 0x076a3, 0x096d0, 0x04afb, 0x04ad0, 0x0a4d0, 0x1d0b6, 0x0d250, 0x0d520, 0x0dd45,
		0x0b5a0, 0x056d0, 0x055b2, 0x049b0, 0x0a577, 0x0a4b0, 0x0aa50, 0x1b255, 0x06d20, 0x0ada0,
		0x14b63, 0x09370, 0x049f8, 0x04970, 0x064b0, 0x168a6, 0x0ea50, 0x06aa0, 0x1a6c4, 0x0aae0,
		0x092e0, 0x0d2e3, 0x0c960, 0x0d557, 0x0d4a0, 0x0da50, 0x05d55, 0x056a0, 0x0a6d0, 0x055d4,
		0x052d0, 0x0a9b8, 0x0a950, 0x0b4a0, 0x0b6a6, 0x0ad50, 0x055a0, 0x0aba4, 0x0a5b0, 0x052b0,
		0x0b273, 0x06930, 0x07337, 0x06aa0, 0x0ad50, 0x14b55, 0x04b60, 0x0a570, 0x054e4, 0x0d160,
		0x0e968, 0x0d520, 0x0daa0, 0x16aa6, 0x056d0, 0x04ae0, 0x0a9d4, 0x0a4d0, 0x0d150, 0x0f252,
		0x0d520,
	}

	gYear, gMonth, gDay := t.Year(), int(t.Month()), t.Day()
	baseDate := time.Date(1900, 1, 31, 0, 0, 0, 0, time.UTC)
	targetDate := time.Date(gYear, time.Month(gMonth), gDay, 0, 0, 0, 0, time.UTC)
	offset := int(targetDate.Sub(baseDate).Hours() / 24)

	if offset < 0 {
		return "Lunar calendar available from 1900 onwards"
	}

	lunarYear := 1900
	for i := 0; i < len(lunarInfo) && offset >= 0; i++ {
		daysInYear := 0
		for j := 0; j < 12; j++ {
			if lunarInfo[i]&(0x10000>>j) != 0 {
				daysInYear += 30
			} else {
				daysInYear += 29
			}
		}
		leapMonth := lunarInfo[i] & 0xf
		if leapMonth != 0 {
			if lunarInfo[i]&0x10000 != 0 {
				daysInYear += 30
			} else {
				daysInYear += 29
			}
		}
		if offset < daysInYear {
			break
		}
		offset -= daysInYear
		lunarYear++
	}

	leapMonth := lunarInfo[lunarYear-1900] & 0xf
	isLeapMonth := false
	lunarMonth := 1

	for lunarMonth <= 12 && offset > 0 {
		monthDays := 29
		if lunarInfo[lunarYear-1900]&(0x10000>>(lunarMonth-1)) != 0 {
			monthDays = 30
		}
		if offset < monthDays {
			break
		}
		offset -= monthDays

		if leapMonth != 0 && lunarMonth == leapMonth {
			leapDays := 29
			if lunarInfo[lunarYear-1900]&0x10000 != 0 {
				leapDays = 30
			}
			if offset < leapDays {
				isLeapMonth = true
				break
			}
			offset -= leapDays
		}
		lunarMonth++
	}
	if lunarMonth > 12 {
		lunarMonth = 12
	}

	zodiac := []string{"Monkey", "Rooster", "Dog", "Pig", "Rat", "Ox", "Tiger", "Rabbit", "Dragon", "Snake", "Horse", "Goat"}
	stems := []string{"Jia", "Yi", "Bing", "Ding", "Wu", "Ji", "Geng", "Xin", "Ren", "Gui"}
	branches := []string{"Zi", "Chou", "Yin", "Mao", "Chen", "Si", "Wu", "Wei", "Shen", "You", "Xu", "Hai"}
	ganZhi := fmt.Sprintf("%s-%s", stems[(lunarYear-4)%10], branches[(lunarYear-4)%12])

	monthStr := fmt.Sprintf("%d", lunarMonth)
	if isLeapMonth {
		monthStr = "Leap " + monthStr
	}

	return fmt.Sprintf("Year %d (%s), Month %s, Day %d | Gan-Zhi: %s | Zodiac: %s",
		lunarYear, ganZhi, monthStr, offset+1, ganZhi, zodiac[lunarYear%12])
}

func showAllCalendars() (string, error) {
	now := time.Now()
	gDate := now.Format("2006-01-02")

	isl, _ := gregorianToIslamic(now)
	per := gregorianToPersian(now)
	heb, _ := gregorianToHebrew(now)
	lun := gregorianToChineseLunar(now)

	weekdays := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	weekday := weekdays[now.Weekday()]
	_, isoWeek := now.ISOWeek()

	var sb strings.Builder
	sb.WriteString("========================================\n")
	sb.WriteString("        Current Date & Time\n")
	sb.WriteString("========================================\n")
	sb.WriteString(fmt.Sprintf("Gregorian : %s %s\n", gDate, weekday))
	sb.WriteString(fmt.Sprintf("Time      : %s\n", now.Format("15:04:05")))
	sb.WriteString(fmt.Sprintf("UTC       : %s\n", now.UTC().Format("15:04:05")))
	sb.WriteString(fmt.Sprintf("Unix TS   : %d\n", now.Unix()))
	sb.WriteString(fmt.Sprintf("ISO Week  : %d\n", isoWeek))
	sb.WriteString(fmt.Sprintf("Day of Year: %d\n", now.YearDay()))
	sb.WriteString("----------------------------------------\n")
	sb.WriteString(fmt.Sprintf("Islamic   : %s\n", isl))
	sb.WriteString(fmt.Sprintf("Persian   : %s\n", per))
	sb.WriteString(fmt.Sprintf("Hebrew    : %s\n", heb))
	sb.WriteString(fmt.Sprintf("Chinese   : %s\n", lun))
	sb.WriteString("========================================\n")
	return sb.String(), nil
}

func getHolidays(country, year string) ([]Holiday, error) {
	if year == "" {
		year = time.Now().Format("2006")
	}
	url := fmt.Sprintf("https://date.nager.at/api/v3/PublicHolidays/%s/%s", year, country)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("API failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var holidays []Holiday
	if err := json.Unmarshal(body, &holidays); err != nil {
		return nil, fmt.Errorf("parse error: %v", err)
	}
	return holidays, nil
}

func setReminder(name, date string) error {
	reminderDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("invalid date format: use YYYY-MM-DD")
	}
	if reminderDate.Before(time.Now()) {
		return fmt.Errorf("reminder date is in the past")
	}
	duration := reminderDate.Sub(time.Now())
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	fmt.Printf("Reminder '%s' set for %s\n", name, date)
	fmt.Printf("Time until: %d days %d hours\n", days, hours)
	fmt.Println("(Persistence support requires file storage)")
	return nil
}
