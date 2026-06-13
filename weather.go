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

type OpenMeteoCurrent struct {
	Temperature2M      float64 `json:"temperature_2m"`
	RelativeHumidity2M float64 `json:"relative_humidity_2m"`
	ApparentTemperature float64 `json:"apparent_temperature"`
	Precipitation      float64 `json:"precipitation"`
	WeatherCode        int     `json:"weather_code"`
	CloudCover         float64 `json:"cloud_cover"`
	PressureMSL        float64 `json:"pressure_msl"`
	WindSpeed10M       float64 `json:"wind_speed_10m"`
	WindDirection10M   float64 `json:"wind_direction_10m"`
	UVIndex            float64 `json:"uv_index"`
}

func handleWeather(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: weathercli weather <city>")
		fmt.Println("Example: weathercli weather 'New York'")
		fmt.Println("Also try: weathercli feelslike <temp> <humidity> <wind>")
		fmt.Println("         weathercli ntp")
		return
	}
	fmt.Print(getWeather(strings.Join(args, " ")))
}

func handleFeelsLike(args []string) {
	if len(args) < 3 {
		fmt.Println("Usage: weathercli feelslike <temp> <humidity> <wind>")
		fmt.Println("Example: weathercli feelslike 32 70 15")
		return
	}
	temp, _ := strconv.ParseFloat(args[0], 64)
	humidity, _ := strconv.ParseFloat(args[1], 64)
	wind, _ := strconv.ParseFloat(args[2], 64)
	fmt.Print(calculateFeelsLike(temp, humidity, wind))
}

func handleNTP(args []string) {
	fmt.Print(getNTPTime())
}

func getWeather(city string) string {
	var sb strings.Builder

	// Geocode
	geoURL := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?"+
		"name=%s&count=1&language=en&format=json", strings.ReplaceAll(city, " ", "%20"))

	resp, err := http.Get(geoURL)
	if err != nil {
		sb.WriteString(fmt.Sprintf("Geocoding failed: %v\n", err))
		return sb.String()
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	type GeoResult struct {
		Results []struct {
			Name        string  `json:"name"`
			Latitude    float64 `json:"latitude"`
			Longitude   float64 `json:"longitude"`
			Country     string  `json:"country"`
			CountryCode string  `json:"country_code"`
			Timezone    string  `json:"timezone"`
			Admin1      string  `json:"admin1"`
		} `json:"results"`
	}

	var geo GeoResult
	if json.Unmarshal(body, &geo) != nil || len(geo.Results) == 0 {
		sb.WriteString(fmt.Sprintf("City not found: %s\n", city))
		return sb.String()
	}

	loc := geo.Results[0]

	// Fetch weather
	wURL := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?"+
		"latitude=%.4f&longitude=%.4f&current=temperature_2m,relative_humidity_2m,"+
		"apparent_temperature,precipitation,weather_code,cloud_cover,pressure_msl,"+
		"wind_speed_10m,wind_direction_10m,uv_index&daily=temperature_2m_max,"+
		"temperature_2m_min,precipitation_sum,sunrise,sunset&timezone=auto&forecast_days=3",
		loc.Latitude, loc.Longitude)

	wResp, err := http.Get(wURL)
	if err != nil {
		sb.WriteString(fmt.Sprintf("Weather API failed: %v\n", err))
		return sb.String()
	}
	defer wResp.Body.Close()

	body, _ = io.ReadAll(wResp.Body)
	var weatherData struct {
		Current OpenMeteoCurrent `json:"current"`
		Daily   struct {
			Time             []string  `json:"time"`
			Temperature2MMax []float64 `json:"temperature_2m_max"`
			Temperature2MMin []float64 `json:"temperature_2m_min"`
			PrecipitationSum []float64 `json:"precipitation_sum"`
			Sunrise          []string  `json:"sunrise"`
			Sunset           []string  `json:"sunset"`
		} `json:"daily"`
	}

	if json.Unmarshal(body, &weatherData) != nil {
		sb.WriteString("Failed to parse weather data.\n")
		return sb.String()
	}

	current := weatherData.Current

	tz, _ := time.LoadLocation(loc.Timezone)
	if tz == nil {
		tz = time.UTC
	}

	sb.WriteString("========================================\n")
	sb.WriteString(fmt.Sprintf("  Weather: %s, %s (%s)\n", loc.Name, loc.Country, loc.Admin1))
	sb.WriteString(fmt.Sprintf("  Location: %.4f, %.4f\n", loc.Latitude, loc.Longitude))
	sb.WriteString("========================================\n")
	sb.WriteString(fmt.Sprintf("Condition    : %s\n", getWeatherCondition(current.WeatherCode)))
	sb.WriteString(fmt.Sprintf("Temperature  : %.1f°C (Feels %.1f°C)\n",
		current.Temperature2M, current.ApparentTemperature))
	sb.WriteString(fmt.Sprintf("Humidity     : %.0f%%\n", current.RelativeHumidity2M))
	sb.WriteString(fmt.Sprintf("Wind         : %.1f km/h %s\n",
		current.WindSpeed10M, getWindDirection(current.WindDirection10M)))
	sb.WriteString(fmt.Sprintf("Pressure     : %.1f hPa\n", current.PressureMSL))
	sb.WriteString(fmt.Sprintf("Cloud Cover  : %.0f%%\n", current.CloudCover))
	sb.WriteString(fmt.Sprintf("Precipitation: %.1f mm\n", current.Precipitation))
	sb.WriteString(fmt.Sprintf("UV Index     : %.1f (%s)\n", current.UVIndex, getUVLevel(current.UVIndex)))

	tempDiff := current.ApparentTemperature - current.Temperature2M
	if math.Abs(tempDiff) > 2 {
		if tempDiff > 0 {
			sb.WriteString(fmt.Sprintf("  💡 Feels %.1f°C warmer (humidity effect)\n", tempDiff))
		} else {
			sb.WriteString(fmt.Sprintf("  💡 Feels %.1f°C colder (wind chill)\n", -tempDiff))
		}
	}

	currentTime := time.Now().In(tz)
	sb.WriteString(fmt.Sprintf("Local Time   : %s\n", currentTime.Format("2006-01-02 15:04")))

	// 3-day forecast
	sb.WriteString("\n  --- 3-Day Forecast ---\n")
	for i, day := range weatherData.Daily.Time {
		t, _ := time.Parse("2006-01-02", day)
		sb.WriteString(fmt.Sprintf("  %s (%s): %.1f~%.1f°C, %.1fmm\n",
			day, t.Format("Mon"),
			weatherData.Daily.Temperature2MMin[i],
			weatherData.Daily.Temperature2MMax[i],
			weatherData.Daily.PrecipitationSum[i]))
	}

	if len(weatherData.Daily.Sunrise) > 0 && len(weatherData.Daily.Sunrise[0]) >= 16 {
		sr := weatherData.Daily.Sunrise[0]
		ss := weatherData.Daily.Sunset[0]
		sb.WriteString(fmt.Sprintf("  Sunrise: %s | Sunset: %s\n",
			sr[len(sr)-5:], ss[len(ss)-5:]))
	}

	sb.WriteString("========================================\n")
	return sb.String()
}

func getWeatherCondition(code int) string {
	conditions := map[int]string{
		0: "Clear Sky", 1: "Mainly Clear", 2: "Partly Cloudy", 3: "Overcast",
		45: "Foggy", 48: "Rime Fog",
		51: "Light Drizzle", 53: "Moderate Drizzle", 55: "Dense Drizzle",
		56: "Freezing Drizzle", 57: "Freezing Drizzle",
		61: "Slight Rain", 63: "Moderate Rain", 65: "Heavy Rain",
		66: "Freezing Rain", 67: "Freezing Rain",
		71: "Slight Snow", 73: "Moderate Snow", 75: "Heavy Snow", 77: "Snow Grains",
		80: "Rain Showers", 81: "Moderate Rain", 82: "Violent Rain",
		85: "Snow Showers", 86: "Heavy Snow",
		95: "Thunderstorm", 96: "Thunderstorm+Hail", 99: "Thunderstorm+Hail",
	}
	if c, ok := conditions[code]; ok {
		return c
	}
	return fmt.Sprintf("Code %d", code)
}

func getWindDirection(deg float64) string {
	dirs := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE",
		"S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	return dirs[int(math.Round(deg/22.5))%16]
}

func getUVLevel(uv float64) string {
	switch {
	case uv <= 2:
		return "Low"
	case uv <= 5:
		return "Moderate"
	case uv <= 7:
		return "High"
	case uv <= 10:
		return "Very High"
	default:
		return "Extreme"
	}
}

func getNTPTime() string {
	var sb strings.Builder
	sb.WriteString("========================================\n")
	sb.WriteString("  Precise Time Information\n")
	sb.WriteString("========================================\n")

	now := time.Now()
	sb.WriteString(fmt.Sprintf("System Time  : %s\n", now.Format(time.RFC1123)))
	sb.WriteString(fmt.Sprintf("Unix (s)     : %d\n", now.Unix()))
	sb.WriteString(fmt.Sprintf("Unix (ms)    : %d\n", now.UnixMilli()))
	sb.WriteString(fmt.Sprintf("Unix (ns)    : %d\n", now.UnixNano()))

	// Network time
	urls := []string{
		"https://worldtimeapi.org/api/ip",
		"https://timeapi.io/api/Time/current/zone?timeZone=UTC",
	}
	for _, url := range urls {
		if resp, err := http.Get(url); err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			var result map[string]interface{}
			if json.Unmarshal(body, &result) == nil {
				if dt, ok := result["datetime"]; ok {
					sb.WriteString(fmt.Sprintf("Network Time : %s\n", dt))
					break
				}
				if dt, ok := result["dateTime"]; ok {
					sb.WriteString(fmt.Sprintf("Network Time : %s\n", dt))
					break
				}
			}
		}
	}

	sb.WriteString(fmt.Sprintf("UTC Time     : %s\n", now.UTC().Format(time.RFC1123)))
	_, offset := now.Zone()
	sb.WriteString(fmt.Sprintf("Timezone     : %s (UTC%+d)\n", now.Format("MST"), offset/3600))
	sb.WriteString("========================================\n")
	return sb.String()
}
