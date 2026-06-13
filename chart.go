package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func handleChart(args []string) {
	if len(args) < 1 {
		fmt.Println("Chart subcommands: weather, air, sample")
		fmt.Println("Run 'weathercli help' for details")
		return
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "weather":
		if len(subArgs) < 1 {
			fmt.Println("Usage: weathercli chart weather <city> [days]")
			return
		}
		days := "7"
		if len(subArgs) > 1 {
			days = subArgs[1]
		}
		fmt.Print(chartWeatherTrend(subArgs[0], days))

	case "air":
		if len(subArgs) < 1 {
			fmt.Println("Usage: weathercli chart air <city> [days]")
			return
		}
		days := "7"
		if len(subArgs) > 1 {
			days = subArgs[1]
		}
		fmt.Print(chartAirQuality(subArgs[0], days))

	case "sample":
		fmt.Print(chartSample())

	default:
		fmt.Printf("Unknown chart subcommand: %s\n", sub)
	}
}

func chartWeatherTrend(city string, days string) string {
	var sb strings.Builder

	numDays, err := strconv.Atoi(days)
	if err != nil || numDays < 1 || numDays > 30 {
		numDays = 7
	}

	// Geocode
	geoURL := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?"+
		"name=%s&count=1&language=en&format=json", strings.ReplaceAll(city, " ", "%20"))

	resp, err := http.Get(geoURL)
	if err != nil {
		sb.WriteString(fmt.Sprintf("Error: %v\n", err))
		return sb.String()
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var geo struct {
		Results []struct {
			Name      string  `json:"name"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Country   string  `json:"country"`
		} `json:"results"`
	}

	if json.Unmarshal(body, &geo) != nil || len(geo.Results) == 0 {
		sb.WriteString("City not found.\n")
		return sb.String()
	}

	loc := geo.Results[0]
	startDate := time.Now().AddDate(0, 0, -numDays+1).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")

	// Get historical weather
	wURL := fmt.Sprintf("https://archive-api.open-meteo.com/v1/archive?"+
		"latitude=%.4f&longitude=%.4f&start_date=%s&end_date=%s&"+
		"daily=temperature_2m_max,temperature_2m_min&timezone=auto",
		loc.Latitude, loc.Longitude, startDate, endDate)

	wResp, err := http.Get(wURL)
	if err != nil {
		sb.WriteString(fmt.Sprintf("Error fetching data: %v\n", err))
		return sb.String()
	}
	defer wResp.Body.Close()

	body, _ = io.ReadAll(wResp.Body)
	var data struct {
		Daily struct {
			Time           []string  `json:"time"`
			TemperatureMax []float64 `json:"temperature_2m_max"`
			TemperatureMin []float64 `json:"temperature_2m_min"`
		} `json:"daily"`
	}

	if json.Unmarshal(body, &data) != nil || len(data.Daily.Time) == 0 {
		sb.WriteString("No data available.\n")
		return sb.String()
	}

	// Compute avg temps
	avgTemps := make([]float64, len(data.Daily.Time))
	labels := make([]string, len(data.Daily.Time))
	for i := range data.Daily.Time {
		avgTemps[i] = (data.Daily.TemperatureMax[i] + data.Daily.TemperatureMin[i]) / 2
		t, _ := time.Parse("2006-01-02", data.Daily.Time[i])
		labels[i] = t.Format("Jan02")
	}

	sb.WriteString("========================================\n")
	sb.WriteString(fmt.Sprintf("  Temperature: %s, %s\n", loc.Name, loc.Country))
	sb.WriteString(fmt.Sprintf("  %s to %s\n", startDate, endDate))
	sb.WriteString("========================================\n")

	sb.WriteString(renderASCIIChart(avgTemps, 12, 50, "°C"))
	sb.WriteString("\n  ")
	for i, l := range labels {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(l)
	}
	sb.WriteString("\n========================================\n")
	return sb.String()
}

func chartAirQuality(city string, days string) string {
	var sb strings.Builder

	numDays, _ := strconv.Atoi(days)
	if numDays < 1 || numDays > 30 {
		numDays = 7
	}

	geoURL := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?"+
		"name=%s&count=1&language=en&format=json", strings.ReplaceAll(city, " ", "%20"))

	resp, err := http.Get(geoURL)
	if err != nil {
		sb.WriteString(fmt.Sprintf("Error: %v\n", err))
		return sb.String()
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var geo struct {
		Results []struct {
			Name      string  `json:"name"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Country   string  `json:"country"`
		} `json:"results"`
	}

	if json.Unmarshal(body, &geo) != nil || len(geo.Results) == 0 {
		sb.WriteString("City not found.\n")
		return sb.String()
	}

	loc := geo.Results[0]
	startDate := time.Now().AddDate(0, 0, -numDays+1).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")

	aqURL := fmt.Sprintf("https://air-quality-api.open-meteo.com/v1/air-quality?"+
		"latitude=%.4f&longitude=%.4f&start_date=%s&end_date=%s&"+
		"hourly=us_aqi&timezone=auto",
		loc.Latitude, loc.Longitude, startDate, endDate)

	aqResp, err := http.Get(aqURL)
	if err != nil {
		sb.WriteString(fmt.Sprintf("Error fetching AQI: %v\n", err))
		return sb.String()
	}
	defer aqResp.Body.Close()

	body, _ = io.ReadAll(aqResp.Body)
	var data struct {
		Hourly struct {
			UsAQI []float64 `json:"us_aqi"`
		} `json:"hourly"`
	}

	sb.WriteString("========================================\n")
	sb.WriteString(fmt.Sprintf("  Air Quality: %s, %s\n", loc.Name, loc.Country))
	sb.WriteString("========================================\n")

	if json.Unmarshal(body, &data) != nil || len(data.Hourly.UsAQI) == 0 {
		sb.WriteString("AQI data unavailable. Try a larger city.\n")
		sb.WriteString("\nSample chart:\n")
		sb.WriteString(chartSample())
		return sb.String()
	}

	// Sample every 6 hours
	var sampled []float64
	hasData := false
	for i := 0; i < len(data.Hourly.UsAQI); i += 6 {
		v := data.Hourly.UsAQI[i]
		if !math.IsNaN(v) && v > 0 {
			sampled = append(sampled, v)
			hasData = true
		}
	}

	if !hasData {
		sb.WriteString("No valid AQI data.\n")
		return sb.String()
	}

	sb.WriteString(renderASCIIChart(sampled, 10, 50, "AQI"))
	sb.WriteString(fmt.Sprintf("\n  Data pts: %d (6h intervals)\n", len(sampled)))
	sb.WriteString("  Good(0-50) Moderate(51-100) Unhealthy(101+)\n")
	sb.WriteString("========================================\n")
	return sb.String()
}

func chartSample() string {
	data := []float64{120, 95, 80, 65, 55, 70, 85, 100, 90, 75, 60, 50,
		45, 55, 70, 88, 110, 95, 82, 68, 58, 48, 42, 52}

	var sb strings.Builder
	sb.WriteString("========================================\n")
	sb.WriteString("  Sample ASCII Chart\n")
	sb.WriteString("========================================\n")
	sb.WriteString(renderASCIIChart(data, 12, 50, "Value"))
	sb.WriteString(fmt.Sprintf("\n  Points: %d\n", len(data)))
	sb.WriteString("========================================\n")
	return sb.String()
}

func renderASCIIChart(data []float64, height, width int, label string) string {
	if len(data) == 0 {
		return "[no data]\n"
	}

	minVal, maxVal := data[0], data[0]
	for _, v := range data {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	if maxVal == minVal {
		maxVal = minVal + 10
	}

	range_ := maxVal - minVal
	if range_ == 0 {
		range_ = 1
	}

	// Build the chart
	var result strings.Builder
	result.WriteString(fmt.Sprintf("  %s: %.0f - %.0f\n", label, minVal, maxVal))

	cols := len(data)
	if cols > width-4 {
		cols = width - 4
	}

	step := float64(len(data)) / float64(cols)
	if step < 1 {
		step = 1
	}

	// Sample data to fit width
	sampled := make([]float64, cols)
	for i := 0; i < cols; i++ {
		idx := int(float64(i) * step)
		if idx >= len(data) {
			idx = len(data) - 1
		}
		sampled[i] = data[idx]
	}

	// Render rows
	for row := height; row >= 0; row-- {
		if row == height {
			result.WriteString(fmt.Sprintf("  %.0f ┤", maxVal))
		} else if row == 0 {
			result.WriteString(fmt.Sprintf("  %.0f ┤", minVal))
		} else {
			result.WriteString("     │")
		}

		for _, v := range sampled {
			rowVal := minVal + range_*float64(row)/float64(height)
			nextRowVal := minVal + range_*float64(row-1)/float64(height)

			if (v >= rowVal && v < nextRowVal) || (row == 0 && v <= rowVal) || (row == height && v >= rowVal) {
				result.WriteString("█")
			} else if v >= rowVal {
				result.WriteString("▓")
			} else if row > 0 && v >= nextRowVal-range_/float64(height)/2 {
				result.WriteString("░")
			} else {
				result.WriteString(" ")
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}
