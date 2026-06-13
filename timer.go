package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func handleTimer(args []string) {
	if len(args) < 1 {
		fmt.Println("Timer subcommands: pomodoro, countdown, stopwatch, alarm")
		fmt.Println("Run 'weathercli help' for details")
		return
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "pomodoro":
		workMin := 25
		breakMin := 5
		if len(subArgs) > 0 {
			if v, err := strconv.Atoi(subArgs[0]); err == nil {
				workMin = v
			}
		}
		if len(subArgs) > 1 {
			if v, err := strconv.Atoi(subArgs[1]); err == nil {
				breakMin = v
			}
		}
		fmt.Printf("Starting Pomodoro: %d min work, %d min break\n", workMin, breakMin)
		startPomodoro(workMin, breakMin)

	case "countdown":
		if len(subArgs) < 1 {
			fmt.Println("Usage: weathercli timer countdown <seconds>")
			return
		}
		seconds, err := strconv.Atoi(subArgs[0])
		if err != nil {
			fmt.Printf("Invalid seconds: %s\n", subArgs[0])
			return
		}
		startCountdown(seconds)

	case "stopwatch":
		startStopwatch()

	case "alarm":
		if len(subArgs) < 1 {
			fmt.Println("Usage: weathercli timer alarm <HH:MM>")
			return
		}
		setAlarm(subArgs[0])

	default:
		fmt.Printf("Unknown timer subcommand: %s\n", sub)
	}
}

func startPomodoro(workMin, breakMin int) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	session := 1
	running := true

	for running {
		select {
		case <-sigCh:
			fmt.Println("\nPomodoro stopped by user.")
			return
		default:
		}

		fmt.Printf("\n=== 🍅 Session %d: Work (%d min) ===\n", session, workMin)
		if !countdownWithSignal(time.Duration(workMin)*time.Minute, sigCh) {
			fmt.Println("\nPomodoro stopped.")
			return
		}
		fmt.Println("\n✅ Work session complete! Time for a break.")
		bell()

		select {
		case <-sigCh:
			fmt.Println("\nPomodoro stopped.")
			return
		default:
		}

		fmt.Printf("\n=== ☕ Break (%d min) ===\n", breakMin)
		if !countdownWithSignal(time.Duration(breakMin)*time.Minute, sigCh) {
			fmt.Println("\nPomodoro stopped.")
			return
		}
		fmt.Println("\n✅ Break complete! Ready for next session.")
		bell()

		session++
	}
}

func startCountdown(seconds int) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("Countdown: %d seconds\n", seconds)

	if !countdownWithSignal(time.Duration(seconds)*time.Second, sigCh) {
		fmt.Println("\nCountdown stopped.")
		return
	}

	fmt.Println("\n⏰ TIME'S UP!")
	for i := 0; i < 3; i++ {
		fmt.Print("\a")
		time.Sleep(300 * time.Millisecond)
	}
}

func startStopwatch() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Stopwatch started. Press Ctrl+C to stop.")
	start := time.Now()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			elapsed := time.Since(start)
			fmt.Printf("\n\nStopwatch stopped.\n")
			fmt.Printf("Elapsed: %s\n", formatDuration(elapsed))
			fmt.Printf("Seconds: %.2f\n", elapsed.Seconds())
			return
		case <-ticker.C:
			elapsed := time.Since(start)
			fmt.Printf("\rElapsed: %s", formatDuration(elapsed))
		}
	}
}

func setAlarm(timeStr string) {
	now := time.Now()
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		fmt.Printf("Invalid time: %s (use HH:MM 24h)\n", timeStr)
		return
	}

	alarmTime := time.Date(now.Year(), now.Month(), now.Day(),
		t.Hour(), t.Minute(), 0, 0, now.Location())
	if alarmTime.Before(now) {
		alarmTime = alarmTime.AddDate(0, 0, 1)
	}

	duration := alarmTime.Sub(now)
	fmt.Printf("Alarm set for %s\n", alarmTime.Format("15:04"))
	fmt.Printf("Time remaining: %s\n", formatDuration(duration))

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			fmt.Println("\nAlarm cancelled.")
			return
		case <-ticker.C:
			remaining := alarmTime.Sub(time.Now())
			if remaining <= 0 {
				fmt.Println("\n⏰ ALARM! ⏰")
				for i := 0; i < 10; i++ {
					fmt.Print("\a")
					time.Sleep(300 * time.Millisecond)
				}
				fmt.Println("Alarm triggered at", alarmTime.Format("15:04"))
				return
			}
			fmt.Printf("\rTime until alarm: %s", formatDuration(remaining))
		}
	}
}

func countdownWithSignal(duration time.Duration, sigCh chan os.Signal) bool {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	start := time.Now()
	remaining := duration

	for remaining > 0 {
		select {
		case <-sigCh:
			return false
		case <-ticker.C:
			remaining = duration - time.Since(start)
			if remaining < 0 {
				remaining = 0
			}
			fmt.Printf("\r%s remaining", formatDuration(remaining))
		}
	}
	fmt.Print(strings.Repeat(" ", 50) + "\r")
	return true
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func bell() {
	fmt.Print("\a")
}
