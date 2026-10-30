// budget prints what an SLO actually allows: downtime, bad requests,
// bad timeslices, and how long each burn rate takes to empty the budget.
//
//	go run . -slo 0.998 -window 28d -slice 15m -rps 20
package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func main() {
	slo := flag.Float64("slo", 0.998, "SLO target, e.g. 0.998")
	windowFlag := flag.String("window", "28d", "SLO window (supports d suffix)")
	sliceFlag := flag.Duration("slice", 15*time.Minute, "timeslice length")
	rps := flag.Float64("rps", 20, "average requests per second")
	flag.Parse()

	window, err := parseDays(*windowFlag)
	if err != nil {
		fmt.Println("bad -window:", err)
		return
	}

	budget := 1 - *slo
	fmt.Printf("SLO %.3f%% over %s → error budget %.3f%%\n\n", *slo*100, *windowFlag, budget*100)

	fmt.Println("Request-based SLO")
	total := *rps * window.Seconds()
	fmt.Printf("  total requests     %.0f\n", total)
	fmt.Printf("  allowed failures   %.0f\n", total*budget)
	fmt.Printf("  = full outage of   %s\n\n", round(time.Duration(float64(window)*budget)))

	fmt.Println("Timeslice SLO")
	slices := float64(window / *sliceFlag)
	fmt.Printf("  slices of %s     %.0f\n", *sliceFlag, slices)
	fmt.Printf("  allowed bad slices %.2f (≈ %s of bad slices)\n\n", slices*budget,
		round(time.Duration(slices*budget)*(*sliceFlag)))

	fmt.Println("Burn rate → time to empty the whole budget")
	for _, br := range []float64{1, 2, 6, 14.4, 36} {
		fmt.Printf("  %5.1fx  %s\n", br, round(time.Duration(float64(window)/br)))
	}

	fmt.Println("\nBudget spent by a burn rate sustained over an alert window")
	for _, a := range []struct {
		br  float64
		dur time.Duration
	}{{14.4, time.Hour}, {6, 6 * time.Hour}, {1, 3 * 24 * time.Hour}} {
		fmt.Printf("  %5.1fx for %-4s → %.1f%% of budget\n", a.br, short(a.dur), a.br*float64(a.dur)/float64(window)*100)
	}
}

// parseDays accepts Go durations plus a "d" (day) suffix.
func parseDays(s string) (time.Duration, error) {
	if d, ok := strings.CutSuffix(s, "d"); ok {
		n, err := strconv.ParseFloat(d, 64)
		return time.Duration(n * 24 * float64(time.Hour)), err
	}
	return time.ParseDuration(s)
}

func round(d time.Duration) string { return short(d.Round(time.Second)) }

func short(d time.Duration) string {
	if d >= 48*time.Hour {
		return fmt.Sprintf("%.1fd", d.Hours()/24)
	}
	return d.String()
}
