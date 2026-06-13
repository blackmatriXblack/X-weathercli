package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func handleSpace(args []string) {
	if len(args) < 1 {
		fmt.Println("Space subcommands: solar, planet, events, iss, epic")
		fmt.Println("Run 'weathercli help' for details")
		return
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "solar":
		fmt.Print(getSolarActivity())

	case "planet":
		if len(subArgs) < 1 {
			fmt.Println("Usage: weathercli space planet <name>")
			fmt.Println("Planets: mercury, venus, mars, jupiter, saturn, uranus, neptune")
			return
		}
		result, err := getPlanetData(subArgs[0])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Print(result)

	case "events":
		fmt.Print(getAstroEvents())

	case "iss":
		result, err := getISSLocation()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Print(result)

	case "epic":
		result, err := getEPICData()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Print(result)

	default:
		fmt.Printf("Unknown space subcommand: %s\n", sub)
	}
}

func getSolarActivity() string {
	var sb strings.Builder
	sb.WriteString("========================================\n")
	sb.WriteString("     Solar Activity & Space Weather\n")
	sb.WriteString("========================================\n")

	endDate := time.Now().Format("2006-01-02")
	startDate := time.Now().AddDate(0, 0, -7).Format("2006-01-02")

	// Solar flares from NASA DONKI
	url := fmt.Sprintf("https://api.nasa.gov/DONKI/FLR?startDate=%s&endDate=%s&api_key=DEMO_KEY",
		startDate, endDate)
	if resp, err := http.Get(url); err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var flares []struct {
			ClassType string `json:"classType"`
			BeginTime string `json:"beginTime"`
		}
		if json.Unmarshal(body, &flares) == nil {
			if len(flares) == 0 {
				sb.WriteString("No solar flares in last 7 days.\n")
			} else {
				sb.WriteString(fmt.Sprintf("Solar Flares: %d events\n", len(flares)))
				for i, f := range flares {
					if i >= 5 {
						break
					}
					if len(f.BeginTime) >= 10 {
						sb.WriteString(fmt.Sprintf("  %s: %s\n", f.ClassType, f.BeginTime[:10]))
					}
				}
			}
		}
	} else {
		sb.WriteString("Solar data: API unavailable (check network)\n")
	}

	// Solar wind from NOAA
	if resp, err := http.Get("https://services.swpc.noaa.gov/products/solar-wind/plasma-1-day.json"); err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var data [][]interface{}
		if json.Unmarshal(body, &data) == nil && len(data) > 1 {
			latest := data[len(data)-1]
			if len(latest) >= 4 {
				sb.WriteString(fmt.Sprintf("Solar Wind  : %.0f km/s\n", latest[2]))
				sb.WriteString(fmt.Sprintf("Density     : %.1f p/cc\n", latest[3]))
			}
		}
	}

	// Sunspots
	if resp, err := http.Get("https://services.swpc.noaa.gov/products/solar-cycle/sunspots.json"); err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var data [][]interface{}
		if json.Unmarshal(body, &data) == nil && len(data) > 1 {
			latest := data[len(data)-1]
			if len(latest) >= 2 {
				sb.WriteString(fmt.Sprintf("Sunspots    : %.0f\n", latest[1]))
			}
		}
	}

	// Geomagnetic storm data
	geoURL := fmt.Sprintf("https://api.nasa.gov/DONKI/GST?startDate=%s&endDate=%s&api_key=DEMO_KEY",
		startDate, endDate)
	if resp, err := http.Get(geoURL); err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var storms []struct {
			StartTime string `json:"startTime"`
		}
		if json.Unmarshal(body, &storms) == nil {
			sb.WriteString(fmt.Sprintf("Geomagnetic Storms: %d\n", len(storms)))
		}
	}

	sb.WriteString("========================================\n")
	sb.WriteString("Data: NASA DONKI, NOAA SWPC\n")
	sb.WriteString("========================================\n")
	return sb.String()
}

func getPlanetData(planet string) (string, error) {
	planet = strings.ToLower(planet)
	planets := map[string]map[string]string{
		"mercury": {
			"Name": "Mercury", "Diameter": "4,879 km", "DistFromSun": "57.9M km",
			"Orbit": "88 days", "DayLength": "58.6 Earth days",
			"Temperature": "-180 to 430°C", "Atmosphere": "Virtually none",
			"Moons": "0", "FunFact": "Smallest planet; closest to Sun",
			"Gravity": "3.7 m/s²", "Tilt": "0.034°",
		},
		"venus": {
			"Name": "Venus", "Diameter": "12,104 km", "DistFromSun": "108.2M km",
			"Orbit": "225 days", "DayLength": "243 Earth days",
			"Temperature": "462°C avg", "Atmosphere": "96.5% CO2, sulfuric acid",
			"Moons": "0", "FunFact": "Hottest planet; rotates backwards",
			"Gravity": "8.87 m/s²", "Tilt": "177.4°",
		},
		"mars": {
			"Name": "Mars", "Diameter": "6,779 km", "DistFromSun": "227.9M km",
			"Orbit": "687 days", "DayLength": "24.6 hours",
			"Temperature": "-87 to -5°C", "Atmosphere": "95% CO2, 2.7% N2",
			"Moons": "2 (Phobos, Deimos)", "FunFact": "Olympus Mons - largest volcano",
			"Gravity": "3.72 m/s²", "Tilt": "25.2°",
		},
		"jupiter": {
			"Name": "Jupiter", "Diameter": "139,820 km", "DistFromSun": "778.5M km",
			"Orbit": "11.86 years", "DayLength": "9.93 hours",
			"Temperature": "-108°C cloud top", "Atmosphere": "89.8% H2, 10.2% He",
			"Moons": "95 known", "FunFact": "Great Red Spot: 400-year storm",
			"Gravity": "24.79 m/s²", "Tilt": "3.13°",
		},
		"saturn": {
			"Name": "Saturn", "Diameter": "116,460 km", "DistFromSun": "1.43B km",
			"Orbit": "29.46 years", "DayLength": "10.7 hours",
			"Temperature": "-139°C cloud top", "Atmosphere": "96.3% H2, 3.25% He",
			"Moons": "146 known", "FunFact": "Least dense; iconic ring system",
			"Gravity": "10.44 m/s²", "Tilt": "26.73°",
		},
		"uranus": {
			"Name": "Uranus", "Diameter": "50,724 km", "DistFromSun": "2.87B km",
			"Orbit": "84 years", "DayLength": "17.2 hours",
			"Temperature": "-197°C", "Atmosphere": "82.5% H2, 15.2% He",
			"Moons": "27 known", "FunFact": "Rotates on its side (98° tilt)",
			"Gravity": "8.69 m/s²", "Tilt": "97.77°",
		},
		"neptune": {
			"Name": "Neptune", "Diameter": "49,244 km", "DistFromSun": "4.50B km",
			"Orbit": "165 years", "DayLength": "16.1 hours",
			"Temperature": "-201°C", "Atmosphere": "80% H2, 19% He",
			"Moons": "16 known", "FunFact": "Windiest: 2,100 km/h winds",
			"Gravity": "11.15 m/s²", "Tilt": "28.32°",
		},
	}

	data, ok := planets[planet]
	if !ok {
		return "", fmt.Errorf("unknown planet: %s", planet)
	}

	var sb strings.Builder
	sb.WriteString("========================================\n")
	sb.WriteString(fmt.Sprintf("  %s - Planetary Report\n", data["Name"]))
	sb.WriteString("========================================\n")
	fields := []string{"Diameter", "DistFromSun", "Orbit", "DayLength", "Temperature",
		"Atmosphere", "Moons", "Gravity", "Tilt", "FunFact"}
	labels := map[string]string{
		"Diameter": "Diameter", "DistFromSun": "Distance from Sun", "Orbit": "Orbit Period",
		"DayLength": "Day Length", "Temperature": "Temperature", "Atmosphere": "Atmosphere",
		"Moons": "Moons", "Gravity": "Gravity", "Tilt": "Axial Tilt", "FunFact": "Fun Fact",
	}
	for _, f := range fields {
		if v, ok := data[f]; ok {
			sb.WriteString(fmt.Sprintf("%-16s: %s\n", labels[f], v))
		}
	}
	sb.WriteString("========================================\n")
	return sb.String(), nil
}

func getAstroEvents() string {
	var sb strings.Builder
	sb.WriteString("========================================\n")
	sb.WriteString("     Upcoming Astronomical Events\n")
	sb.WriteString("========================================\n")

	now := time.Now()

	// Moon phases (approximate)
	for i := 0; i < 365; i++ {
		d := now.AddDate(0, 0, i)
		daysSinceRef := d.Sub(time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)).Hours() / 24
		phase := (daysSinceRef / 29.53058867) - float64(int(daysSinceRef/29.53058867))
		if phase > 0.48 && phase < 0.52 {
			sb.WriteString(fmt.Sprintf("  Full Moon      : %s\n", d.Format("2006-01-02")))
			break
		}
	}
	for i := 0; i < 365; i++ {
		d := now.AddDate(0, 0, i)
		daysSinceRef := d.Sub(time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)).Hours() / 24
		phase := (daysSinceRef / 29.53058867) - float64(int(daysSinceRef/29.53058867))
		if phase < 0.02 || phase > 0.98 {
			sb.WriteString(fmt.Sprintf("  New Moon       : %s\n", d.Format("2006-01-02")))
			break
		}
	}

	sb.WriteString(fmt.Sprintf("  March Equinox  : ~Mar 20 %d\n", now.Year()))
	sb.WriteString(fmt.Sprintf("  June Solstice  : ~Jun 21 %d\n", now.Year()))
	sb.WriteString(fmt.Sprintf("  Sept Equinox   : ~Sep 22 %d\n", now.Year()))
	sb.WriteString(fmt.Sprintf("  Dec Solstice   : ~Dec 21 %d\n", now.Year()))

	sb.WriteString("  --- Meteor Showers (peak) ---\n")
	showers := []struct{ Name, Date string }{
		{"Quadrantids", "~Jan 3-4"}, {"Lyrids", "~Apr 22-23"},
		{"Eta Aquariids", "~May 5-6"}, {"Perseids", "~Aug 12-13"},
		{"Orionids", "~Oct 21-22"}, {"Leonids", "~Nov 17-18"},
		{"Geminids", "~Dec 13-14"}, {"Ursids", "~Dec 22-23"},
	}
	for _, s := range showers {
		sb.WriteString(fmt.Sprintf("    %-15s: %s\n", s.Name, s.Date))
	}

	sb.WriteString("========================================\n")
	sb.WriteString("Dates are approximate. Visit timeanddate.com\n")
	sb.WriteString("for precise astronomical calculations.\n")
	sb.WriteString("========================================\n")
	return sb.String()
}

func getISSLocation() (string, error) {
	resp, err := http.Get("http://api.open-notify.org/iss-now.json")
	if err != nil {
		return "", fmt.Errorf("ISS API failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Timestamp int64 `json:"timestamp"`
		IssPos    struct {
			Latitude  string `json:"latitude"`
			Longitude string `json:"longitude"`
		} `json:"iss_position"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse error: %v", err)
	}

	var sb strings.Builder
	sb.WriteString("========================================\n")
	sb.WriteString("   International Space Station (ISS)\n")
	sb.WriteString("========================================\n")
	sb.WriteString(fmt.Sprintf("Timestamp   : %s\n", time.Unix(result.Timestamp, 0).Format(time.RFC1123)))
	sb.WriteString(fmt.Sprintf("Latitude    : %s\n", result.IssPos.Latitude))
	sb.WriteString(fmt.Sprintf("Longitude   : %s\n", result.IssPos.Longitude))
	sb.WriteString("Speed       : ~28,000 km/h\n")
	sb.WriteString("Altitude    : ~408 km\n")
	sb.WriteString("Crew        : Active (varies)\n")
	sb.WriteString("Orbit       : ~90 minutes/Earth\n")
	sb.WriteString("========================================\n")
	return sb.String(), nil
}

func getEPICData() (string, error) {
	resp, err := http.Get("https://epic.gsfc.nasa.gov/api/natural")
	if err != nil {
		return "", fmt.Errorf("EPIC API failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var data []struct {
		Identifier string `json:"identifier"`
		Caption    string `json:"caption"`
		Image      string `json:"image"`
		Date       string `json:"date"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("parse error: %v", err)
	}

	var sb strings.Builder
	sb.WriteString("========================================\n")
	sb.WriteString("   NASA EPIC - Earth Imaging Camera\n")
	sb.WriteString("========================================\n")
	sb.WriteString(fmt.Sprintf("Available Images : %d\n", len(data)))
	if len(data) > 0 {
		latest := data[0]
		sb.WriteString(fmt.Sprintf("Latest Caption   : %s\n", latest.Caption))
		sb.WriteString(fmt.Sprintf("Image ID         : %s\n", latest.Image))
		sb.WriteString(fmt.Sprintf("Date             : %s\n", latest.Date))
		sb.WriteString(fmt.Sprintf("Image URL        : https://epic.gsfc.nasa.gov/archive/natural/%s/png/%s.png\n",
			strings.ReplaceAll(latest.Date[:10], "-", "/"), latest.Image))
	}
	sb.WriteString("\nDSCOVR satellite at Lagrange Point 1\n")
	sb.WriteString("captures Earth from ~1.5M km away.\n")
	sb.WriteString("========================================\n")
	return sb.String(), nil
}
