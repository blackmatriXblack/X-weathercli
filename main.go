package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var version = "2.0.0"

func main() {
	if len(os.Args) < 2 {
		interactiveMode()
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "help", "--help", "-h":
		printUsage()
	case "version", "--version", "-v":
		fmt.Printf("ChronoWeather v%s\n", version)
		fmt.Println("Universal Terminal Query Tool")
		fmt.Println("No database required - powered by free APIs & Go stdlib")

	case "calendar", "cal":
		handleCalendar(args)
	case "space":
		handleSpace(args)
	case "earth":
		handleEarth(args)
	case "geo":
		handleGeo(args)
	case "country":
		handleCountry(args)
	case "timer":
		handleTimer(args)
	case "weather", "w":
		handleWeather(args)
	case "feelslike", "feels":
		handleFeelsLike(args)
	case "chart":
		handleChart(args)
	case "ntp":
		handleNTP(args)

	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		fmt.Println("Run 'weathercli help' for usage.")
		os.Exit(1)
	}
}

func interactiveMode() {
	fmt.Println("ChronoWeather v" + version + " - Universal Terminal Query Tool")
	fmt.Println("Type 'help' for commands, 'exit' or 'quit' to close.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("weathercli> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" || input == "q" {
			break
		}
		// Re-parse input as command
		os.Args = append([]string{os.Args[0]}, strings.Fields(input)...)
		args := os.Args[1:]
		subArgs := []string{}
		if len(args) > 1 {
			subArgs = args[1:]
		}
		switch args[0] {
		case "help", "--help", "-h":
			printUsage()
		case "version", "--version", "-v":
			fmt.Printf("ChronoWeather v%s\n", version)
		case "calendar", "cal":
			handleCalendar(subArgs)
		case "space":
			handleSpace(subArgs)
		case "earth":
			handleEarth(subArgs)
		case "geo":
			handleGeo(subArgs)
		case "country":
			handleCountry(subArgs)
		case "timer":
			handleTimer(subArgs)
		case "weather", "w":
			handleWeather(subArgs)
		case "feelslike", "feels":
			handleFeelsLike(subArgs)
		case "chart":
			handleChart(subArgs)
		case "ntp":
			handleNTP(subArgs)
		default:
			fmt.Printf("Unknown: %s. Type 'help'\n", args[0])
		}
		fmt.Println()
	}
}

func printUsage() {
	fmt.Println(`ChronoWeather v` + version + ` - Universal Terminal Query Tool
Usage: weathercli <command> [arguments]

Calendar & Time:
  calendar convert <from> <to> <date>   Convert between calendar systems
  calendar now                           Show all calendar systems
  calendar holidays <country> [year]     List public holidays
  calendar remind <name> <date>          Set a reminder
  ntp                                   Get precise network time

Space & Astronomy:
  space solar                           Solar activity & space weather
  space planet <name>                   Planetary data & atmosphere
  space events                          Astronomical events calendar
  space iss                             Track ISS location
  space epic                            NASA EPIC Earth images

Earth & Environment:
  earth quakes [minMag]                 Recent earthquakes
  earth ocean <stationId>               Ocean data (tides, temp, wind)
  earth air <city>                      Air quality index

Geolocation & Countries:
  geo [location]                        Location lookup (ommit for IP)
  country <code>                        Country info (ISO alpha-2)

Productivity:
  timer pomodoro [work] [break]         Pomodoro focus timer
  timer countdown <seconds>             Countdown timer
  timer stopwatch                       Interactive stopwatch
  timer alarm <HH:MM>                   Set an alarm

Weather:
  weather <city>                        Current weather & feels-like
  feelslike <temp> <humidity> <wind>    Calculate feels-like temp
  chart weather <city> [days]           ASCII temperature chart
  chart air <city> [days]               ASCII air quality chart
  chart sample                          Sample ASCII chart

Help:
  help                                  Show this help message
  version                               Show version

Examples:
  weathercli calendar now
  weathercli calendar holidays CN 2026
  weathercli space planet mars
  weathercli earth quakes 4.0
  weathercli weather "New York"
  weathercli geo "Tokyo"
  weathercli timer pomodoro 25 5
  weathercli feelslike 32 70 15
`)
}

func checkArgs(args []string, min, max int) bool {
	if len(args) < min {
		fmt.Println("Error: insufficient arguments")
		return false
	}
	if max >= 0 && len(args) > max {
		fmt.Println("Error: too many arguments")
		return false
	}
	return true
}

func joinArgs(args []string) string {
	return strings.Join(args, " ")
}
