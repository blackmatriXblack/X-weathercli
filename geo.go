package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func handleGeo(args []string) {
	if len(args) < 1 {
		fmt.Print(getLocationByIP())
		return
	}
	fmt.Print(getLocation(strings.Join(args, " ")))
}

func handleCountry(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: weathercli country <code>")
		fmt.Println("Example: weathercli country JP")
		return
	}
	fmt.Print(getCountryInfo(strings.ToUpper(args[0])))
}

func getLocation(query string) string {
	geoURL := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?"+
		"name=%s&count=1&language=en&format=json", strings.ReplaceAll(query, " ", "%20"))

	var sb strings.Builder

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
			Population  int     `json:"population"`
		} `json:"results"`
	}

	var result GeoResult
	if json.Unmarshal(body, &result) != nil || len(result.Results) == 0 {
		sb.WriteString("Location not found.\n")
		return sb.String()
	}

	loc := result.Results[0]

	// Timezone
	tz, err := time.LoadLocation(loc.Timezone)
	if err != nil {
		tz = time.UTC
	}
	localTime := time.Now().In(tz)
	_, offset := localTime.Zone()

	sb.WriteString("========================================\n")
	sb.WriteString(fmt.Sprintf("  Location: %s\n", loc.Name))
	sb.WriteString("========================================\n")
	sb.WriteString(fmt.Sprintf("Latitude    : %.4f\n", loc.Latitude))
	sb.WriteString(fmt.Sprintf("Longitude   : %.4f\n", loc.Longitude))
	sb.WriteString(fmt.Sprintf("Country     : %s (%s)\n", loc.Country, loc.CountryCode))
	sb.WriteString(fmt.Sprintf("Region      : %s\n", loc.Admin1))
	sb.WriteString(fmt.Sprintf("Population  : %d\n", loc.Population))
	sb.WriteString(fmt.Sprintf("Timezone    : %s (UTC%+d)\n", loc.Timezone, offset/3600))
	sb.WriteString(fmt.Sprintf("Local Time  : %s\n", localTime.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("Zone        : %s\n", localTime.Format("MST")))

	// Map URL
	sb.WriteString(fmt.Sprintf("OpenStreetMap: https://www.openstreetmap.org/?mlat=%.4f&mlon=%.4f\n",
		loc.Latitude, loc.Longitude))

	sb.WriteString("========================================\n")
	return sb.String()
}

func getLocationByIP() string {
	resp, err := http.Get("http://ip-api.com/json/")
	if err != nil {
		return fmt.Sprintf("IP geolocation failed: %v\n", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Query       string  `json:"query"`
		Country     string  `json:"country"`
		CountryCode string  `json:"countryCode"`
		RegionName  string  `json:"regionName"`
		City        string  `json:"city"`
		Zip         string  `json:"zip"`
		Lat         float64 `json:"lat"`
		Lon         float64 `json:"lon"`
		Timezone    string  `json:"timezone"`
		Org         string  `json:"org"`
		Isp         string  `json:"isp"`
		As          string  `json:"as"`
	}

	if json.Unmarshal(body, &result) != nil {
		return "Failed to parse IP location.\n"
	}

	tz, err := time.LoadLocation(result.Timezone)
	if err != nil {
		tz = time.UTC
	}
	localTime := time.Now().In(tz)

	var sb strings.Builder
	sb.WriteString("========================================\n")
	sb.WriteString("  Your Current Location (by IP)\n")
	sb.WriteString("========================================\n")
	sb.WriteString(fmt.Sprintf("IP Address  : %s\n", result.Query))
	sb.WriteString(fmt.Sprintf("Country     : %s (%s)\n", result.Country, result.CountryCode))
	sb.WriteString(fmt.Sprintf("Region      : %s\n", result.RegionName))
	sb.WriteString(fmt.Sprintf("City        : %s\n", result.City))
	sb.WriteString(fmt.Sprintf("Postal Code : %s\n", result.Zip))
	sb.WriteString(fmt.Sprintf("Coordinates : %.4f, %.4f\n", result.Lat, result.Lon))
	sb.WriteString(fmt.Sprintf("Timezone    : %s\n", result.Timezone))
	sb.WriteString(fmt.Sprintf("Local Time  : %s\n", localTime.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("ISP         : %s\n", result.Org))
	sb.WriteString(fmt.Sprintf("ASN         : %s\n", result.As))
	sb.WriteString("========================================\n")
	return sb.String()
}

func getCountryInfo(code string) string {
	url := fmt.Sprintf("https://restcountries.com/v3.1/alpha/%s", code)

	var sb strings.Builder

	resp, err := http.Get(url)
	if err != nil {
		sb.WriteString(fmt.Sprintf("API failed: %v\n", err))
		return sb.String()
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var countries []struct {
		Name struct {
			Common   string `json:"common"`
			Official string `json:"official"`
		} `json:"name"`
		Capital   []string `json:"capital"`
		Region    string   `json:"region"`
		Subregion string   `json:"subregion"`
		Population int64   `json:"population"`
		Area      float64  `json:"area"`
		Timezone  []string `json:"timezones"`
		Currencies map[string]struct {
			Name   string `json:"name"`
			Symbol string `json:"symbol"`
		} `json:"currencies"`
		Languages map[string]string `json:"languages"`
		Demonyms  struct {
			Eng struct {
				M string `json:"m"`
			} `json:"eng"`
		} `json:"demonyms"`
		CapitalInfo struct {
			Latlng []float64 `json:"latlng"`
		} `json:"capitalInfo"`
		Maps struct {
			GoogleMaps string `json:"googleMaps"`
		} `json:"maps"`
		Car struct {
			Side string `json:"side"`
		} `json:"car"`
		Flag string `json:"flag"`
	}

	if json.Unmarshal(body, &countries) != nil || len(countries) == 0 {
		sb.WriteString(fmt.Sprintf("Country not found: %s\n", code))
		return sb.String()
	}

	c := countries[0]

	timezone := "N/A"
	if len(c.Timezone) > 0 {
		timezone = c.Timezone[0]
	}

	currencyName, currencySymbol := "N/A", ""
	for _, curr := range c.Currencies {
		currencyName = curr.Name
		currencySymbol = curr.Symbol
		break
	}

	var localTimeStr string
	tz, err := time.LoadLocation(timezone)
	if err == nil {
		localTimeStr = time.Now().In(tz).Format("2006-01-02 15:04:05")
	} else {
		localTimeStr = "N/A"
	}

	// Collect languages
	langs := make([]string, 0, len(c.Languages))
	for _, v := range c.Languages {
		langs = append(langs, v)
	}

	sb.WriteString("========================================\n")
	sb.WriteString(fmt.Sprintf("  %s %s (%s)\n", c.Flag, c.Name.Common, code))
	sb.WriteString("========================================\n")
	sb.WriteString(fmt.Sprintf("Official    : %s\n", c.Name.Official))
	sb.WriteString(fmt.Sprintf("Capital     : %s\n", strings.Join(c.Capital, ", ")))
	if len(c.CapitalInfo.Latlng) >= 2 {
		sb.WriteString(fmt.Sprintf("Capital Coord: %.4f, %.4f\n", c.CapitalInfo.Latlng[0], c.CapitalInfo.Latlng[1]))
	}
	sb.WriteString(fmt.Sprintf("Region      : %s / %s\n", c.Region, c.Subregion))
	sb.WriteString(fmt.Sprintf("Population  : %d\n", c.Population))
	sb.WriteString(fmt.Sprintf("Area        : %.0f km²\n", c.Area))
	sb.WriteString(fmt.Sprintf("Density     : %.1f/km²\n", float64(c.Population)/c.Area))
	sb.WriteString(fmt.Sprintf("Timezone    : %s\n", timezone))
	sb.WriteString(fmt.Sprintf("Local Time  : %s\n", localTimeStr))
	sb.WriteString(fmt.Sprintf("Currency    : %s %s\n", currencySymbol, currencyName))
	sb.WriteString(fmt.Sprintf("Languages   : %s\n", strings.Join(langs, ", ")))
	sb.WriteString(fmt.Sprintf("Demonym     : %s\n", c.Demonyms.Eng.M))
	sb.WriteString(fmt.Sprintf("Driving     : %s side\n", c.Car.Side))
	sb.WriteString(fmt.Sprintf("Google Maps : %s\n", c.Maps.GoogleMaps))
	sb.WriteString("========================================\n")
	return sb.String()
}
