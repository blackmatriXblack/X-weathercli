package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

type USGSQuake struct {
	Properties struct {
		Mag     float64 `json:"mag"`
		Place   string  `json:"place"`
		Time    int64   `json:"time"`
		Tsunami int     `json:"tsunami"`
		URL     string  `json:"url"`
	} `json:"properties"`
}

type USGSResponse struct {
	Features []USGSQuake `json:"features"`
	Metadata struct {
		Count int `json:"count"`
	} `json:"metadata"`
}

func handleEarth(args []string) {
	if len(args) < 1 {
		fmt.Println("Earth subcommands: quakes, ocean, air")
		fmt.Println("Run 'weathercli help' for details")
		return
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "quakes":
		minMag := "2.5"
		if len(subArgs) > 0 {
			minMag = subArgs[0]
		}
		fmt.Print(getEarthquakes(minMag))

	case "ocean":
		if len(subArgs) < 1 {
			fmt.Println("Usage: weathercli earth ocean <stationId>")
			fmt.Println("Example stations: 9414290 (SF), 8518750 (NY), 8724580 (Key West)")
			return
		}
		fmt.Print(getOceanData(subArgs[0]))

	case "air":
		if len(subArgs) < 1 {
			fmt.Println("Usage: weathercli earth air <city>")
			fmt.Println("Example: weathercli earth air 'Beijing'")
			return
		}
		fmt.Print(getAirQuality(strings.Join(subArgs, " ")))

	default:
		fmt.Printf("Unknown earth subcommand: %s\n", sub)
	}
}

func getEarthquakes(minMag string) string {
	url := "https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/2.5_day.geojson"

	var sb strings.Builder
	sb.WriteString("========================================\n")
	sb.WriteString("  Recent Earthquakes (Past 24h)\n")
	sb.WriteString("========================================\n")

	resp, err := http.Get(url)
	if err != nil {
		sb.WriteString(fmt.Sprintf("Error: %v\n", err))
		return sb.String()
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result USGSResponse
	if json.Unmarshal(body, &result) != nil {
		sb.WriteString("Failed to parse earthquake data.\n")
		return sb.String()
	}

	count := 0
	now := time.Now()
	for _, q := range result.Features {
		t := time.UnixMilli(q.Properties.Time)
		if t.After(now.AddDate(0, 0, -1)) && t.Before(now) {
			magStr := fmt.Sprintf("%.1f", q.Properties.Mag)
			sb.WriteString(fmt.Sprintf("  M%s | %s\n", magStr, q.Properties.Place))
			sb.WriteString(fmt.Sprintf("       %s\n", t.Format("2006-01-02 15:04:05")))
			if q.Properties.Tsunami == 1 {
				sb.WriteString("  ⚠️ TSUNAMI WARNING\n")
			}
			count++
		}
	}

	if count == 0 {
		sb.WriteString("  No earthquakes in the past 24 hours.\n")
	}
	sb.WriteString(fmt.Sprintf("  ---\n  Total events: %d in feed\n", result.Metadata.Count))
	sb.WriteString("========================================\n")
	sb.WriteString("Source: USGS Earthquake Hazards Program\n")
	sb.WriteString("========================================\n")
	return sb.String()
}

func getOceanData(stationId string) string {
	var sb strings.Builder
	sb.WriteString("========================================\n")
	sb.WriteString(fmt.Sprintf("  Ocean Data - Station %s\n", stationId))
	sb.WriteString(fmt.Sprintf("  Location: %s\n", getStationName(stationId)))
	sb.WriteString("========================================\n")

	today := time.Now().Format("20060102")
	tomorrow := time.Now().AddDate(0, 0, 2).Format("20060102")

	// Tide predictions
	tideURL := fmt.Sprintf("https://api.tidesandcurrents.noaa.gov/api/prod/datagetter?"+
		"begin_date=%s&end_date=%s&station=%s&product=predictions&datum=MLLW&"+
		"time_zone=gmt&units=metric&format=json", today, tomorrow, stationId)

	if resp, err := http.Get(tideURL); err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var tideData struct {
			Predictions []struct {
				Time  string `json:"t"`
				Value string `json:"v"`
			} `json:"predictions"`
		}
		if json.Unmarshal(body, &tideData) == nil && len(tideData.Predictions) > 0 {
			sb.WriteString("Tide Predictions (upcoming):\n")
			count := 0
			for _, p := range tideData.Predictions {
				t, err := time.Parse("2006-01-02 15:04", p.Time)
				if err == nil && t.After(time.Now()) && count < 6 {
					sb.WriteString(fmt.Sprintf("  %s : %s m\n", t.Format("15:04"), p.Value))
					count++
				}
			}
			if count == 0 {
				for i, p := range tideData.Predictions[:min(6, len(tideData.Predictions))] {
					sb.WriteString(fmt.Sprintf("  %s : %s m\n", p.Time[11:16], p.Value))
					if i >= 5 {
						break
					}
				}
			}
		} else {
			sb.WriteString("Tide data unavailable for this station.\n")
		}
	} else {
		sb.WriteString("Tide API unavailable (check network).\n")
	}

	// Water temperature
	tempURL := fmt.Sprintf("https://api.tidesandcurrents.noaa.gov/api/prod/datagetter?"+
		"date=%s&station=%s&product=water_temperature&datum=MLLW&"+
		"time_zone=gmt&units=metric&format=json", today, stationId)

	if resp, err := http.Get(tempURL); err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var tempData struct {
			Data []struct {
				Value string `json:"v"`
			} `json:"data"`
		}
		if json.Unmarshal(body, &tempData) == nil && len(tempData.Data) > 0 {
			sb.WriteString(fmt.Sprintf("Water Temp   : %s°C\n", tempData.Data[0].Value))
		}
	}

	// Wind
	windURL := fmt.Sprintf("https://api.tidesandcurrents.noaa.gov/api/prod/datagetter?"+
		"date=%s&station=%s&product=wind&datum=MLLW&"+
		"time_zone=gmt&units=metric&format=json", today, stationId)

	if resp, err := http.Get(windURL); err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var windData struct {
			Data []struct {
				Speed string `json:"s"`
				Dir   string `json:"d"`
			} `json:"data"`
		}
		if json.Unmarshal(body, &windData) == nil && len(windData.Data) > 0 {
			latest := windData.Data[len(windData.Data)-1]
			sb.WriteString(fmt.Sprintf("Wind         : %s m/s from %s°\n", latest.Speed, latest.Dir))
		}
	}

	sb.WriteString("========================================\n")
	sb.WriteString("Source: NOAA CO-OPS API\n")
	sb.WriteString("========================================\n")
	return sb.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getStationName(id string) string {
	stations := map[string]string{
		"9414290": "San Francisco, CA", "8518750": "The Battery, NY",
		"8724580": "Key West, FL", "1612340": "Honolulu, HI",
		"8443970": "Boston, MA", "9432780": "Seattle, WA",
		"8770570": "Sabine Pass, TX", "8736897": "Mobile, AL",
		"8656483": "Wilmington, NC", "9410170": "San Diego, CA",
		"8770613": "Galveston, TX", "8722670": "Mayport, FL",
	}
	if name, ok := stations[id]; ok {
		return name
	}
	return "Station " + id
}

func getAirQuality(city string) string {
	var sb strings.Builder
	sb.WriteString("========================================\n")
	sb.WriteString(fmt.Sprintf("  Air Quality - %s\n", city))
	sb.WriteString("========================================\n")

	// Geocode
	geoURL := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1&language=en&format=json",
		strings.ReplaceAll(city, " ", "%20"))

	resp, err := http.Get(geoURL)
	if err != nil {
		sb.WriteString(fmt.Sprintf("Geocoding failed: %v\n", err))
		return sb.String()
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var geoResult struct {
		Results []struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Name      string  `json:"name"`
			Country   string  `json:"country"`
		} `json:"results"`
	}

	if json.Unmarshal(body, &geoResult) != nil || len(geoResult.Results) == 0 {
		sb.WriteString("City not found.\n")
		return sb.String()
	}

	loc := geoResult.Results[0]
	lat := fmt.Sprintf("%.4f", loc.Latitude)
	lon := fmt.Sprintf("%.4f", loc.Longitude)

	// Air quality API
	aqURL := fmt.Sprintf("https://air-quality-api.open-meteo.com/v1/air-quality?"+
		"latitude=%s&longitude=%s&current=european_aqi,us_aqi,pm2_5,pm10,"+
		"nitrogen_dioxide,ozone,sulphur_dioxide,carbon_monoxide", lat, lon)

	aqResp, err := http.Get(aqURL)
	if err != nil {
		sb.WriteString(fmt.Sprintf("Air quality API failed: %v\n", err))
		return sb.String()
	}
	defer aqResp.Body.Close()

	body, _ = io.ReadAll(aqResp.Body)
	var aqData struct {
		Current struct {
			EuropeanAQI float64 `json:"european_aqi"`
			UsAQI       float64 `json:"us_aqi"`
			PM25        float64 `json:"pm2_5"`
			PM10        float64 `json:"pm10"`
			NO2         float64 `json:"nitrogen_dioxide"`
			O3          float64 `json:"ozone"`
			SO2         float64 `json:"sulphur_dioxide"`
			CO          float64 `json:"carbon_monoxide"`
		} `json:"current"`
	}

	if json.Unmarshal(body, &aqData) != nil {
		sb.WriteString("Failed to parse air quality data.\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("Location    : %s, %s (%s, %s)\n", loc.Name, loc.Country, lat, lon))

	if aqData.Current.UsAQI > 0 {
		aqiLabel := getAQILabel(int(aqData.Current.UsAQI))
		sb.WriteString(fmt.Sprintf("US AQI      : %.0f - %s\n", aqData.Current.UsAQI, aqiLabel))
	}
	if aqData.Current.EuropeanAQI > 0 {
		sb.WriteString(fmt.Sprintf("EU AQI      : %.0f\n", aqData.Current.EuropeanAQI))
	}

	sb.WriteString("  --- Pollutants (µg/m³) ---\n")
	if aqData.Current.PM25 > 0 {
		sb.WriteString(fmt.Sprintf("PM2.5       : %.1f\n", aqData.Current.PM25))
	}
	if aqData.Current.PM10 > 0 {
		sb.WriteString(fmt.Sprintf("PM10        : %.1f\n", aqData.Current.PM10))
	}
	if aqData.Current.NO2 > 0 {
		sb.WriteString(fmt.Sprintf("NO2         : %.1f\n", aqData.Current.NO2))
	}
	if aqData.Current.O3 > 0 {
		sb.WriteString(fmt.Sprintf("O3          : %.1f\n", aqData.Current.O3))
	}
	if aqData.Current.SO2 > 0 {
		sb.WriteString(fmt.Sprintf("SO2         : %.1f\n", aqData.Current.SO2))
	}
	if aqData.Current.CO > 0 {
		sb.WriteString(fmt.Sprintf("CO          : %.1f\n", aqData.Current.CO))
	}

	// Advisory
	aqi := int(aqData.Current.UsAQI)
	switch {
	case aqi > 150:
		sb.WriteString("  ⚠️ UNHEALTHY - Limit outdoor activities\n")
	case aqi > 100:
		sb.WriteString("  ⚠️ Sensitive groups: limit outdoor exertion\n")
	case aqi > 50:
		sb.WriteString("  Moderate air quality.\n")
	default:
		sb.WriteString("  Good air quality.\n")
	}

	sb.WriteString("========================================\n")
	return sb.String()
}

func getAQILabel(aqi int) string {
	switch {
	case aqi <= 50:
		return "Good"
	case aqi <= 100:
		return "Moderate"
	case aqi <= 150:
		return "Unhealthy (Sensitive)"
	case aqi <= 200:
		return "Unhealthy"
	case aqi <= 300:
		return "Very Unhealthy"
	default:
		return "Hazardous"
	}
}

func getComfortLevel(temp, humidity float64) string {
	if temp >= 30 && humidity >= 70 {
		return "Very Uncomfortable (Hot & Humid)"
	}
	if temp >= 30 {
		return "Hot"
	}
	if temp >= 25 {
		if humidity >= 60 {
			return "Warm & Humid"
		}
		return "Warm"
	}
	if temp >= 20 {
		return "Very Comfortable"
	}
	if temp >= 15 {
		return "Mild"
	}
	if temp >= 10 {
		return "Cool"
	}
	if temp >= 5 {
		return "Chilly"
	}
	if temp >= 0 {
		return "Cold"
	}
	return "Very Cold"
}

func calculateFeelsLike(temp, humidity, windSpeed float64) string {
	var sb strings.Builder
	sb.WriteString("========================================\n")
	sb.WriteString("  Feels-Like Temperature Calculator\n")
	sb.WriteString("========================================\n")
	sb.WriteString(fmt.Sprintf("Actual Temp : %.1f°C\n", temp))
	sb.WriteString(fmt.Sprintf("Humidity    : %.0f%%\n", humidity))
	sb.WriteString(fmt.Sprintf("Wind Speed  : %.1f km/h\n", windSpeed))

	// Heat Index (temp >= 27°C)
	if temp >= 27 {
		hi := -8.784695 +
			1.61139411*temp +
			2.338549*humidity -
			0.14611605*temp*humidity -
			0.012308094*temp*temp -
			0.016424828*humidity*humidity +
			0.002211732*temp*temp*humidity +
			0.00072546*temp*humidity*humidity -
			0.000003582*temp*temp*humidity*humidity
		sb.WriteString(fmt.Sprintf("Heat Index  : %.1f°C\n", hi))
	}

	// Wind Chill (temp <= 10°C, wind > 4.8 km/h)
	if temp <= 10 && windSpeed > 4.8 {
		wci := 13.12 + 0.6215*temp - 11.37*math.Pow(windSpeed/3.6, 0.16) + 0.3965*temp*math.Pow(windSpeed/3.6, 0.16)
		sb.WriteString(fmt.Sprintf("Wind Chill  : %.1f°C\n", wci))
	}

	// Apparent Temperature
	if temp > 10 {
		e := (humidity / 100) * 6.105 * math.Exp(17.27*temp/(237.7+temp))
		at := temp + 0.33*e - 0.7*windSpeed/3.6 - 4.0
		sb.WriteString(fmt.Sprintf("Apparent    : %.1f°C\n", at))
	}

	sb.WriteString(fmt.Sprintf("Comfort     : %s\n", getComfortLevel(temp, humidity)))

	if temp >= 35 {
		sb.WriteString("  ⚠️ Extreme Heat - Stay hydrated!\n")
	} else if temp <= -10 {
		sb.WriteString("  ⚠️ Extreme Cold - Dress warmly!\n")
	}

	sb.WriteString("========================================\n")
	return sb.String()
}
