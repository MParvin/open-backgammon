// Command benchdiff runs seeded self-play and prints difficulty win rates.
package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"

	"bg/internal/bot"
)

func main() {
	n := 40
	if len(os.Args) > 1 {
		if v, err := strconv.Atoi(os.Args[1]); err == nil && v > 0 {
			n = v
		}
	}
	easy := bot.Easy{Rand: rand.New(rand.NewPCG(1, 2))}
	mid := bot.Intermediate{Rand: rand.New(rand.NewPCG(3, 4))}
	hard := bot.Hard{Rand: rand.New(rand.NewPCG(5, 6))}

	printSeries("Easy vs Intermediate", bot.RunSeries(easy, mid, n, 1001))
	printSeries("Intermediate vs Hard", bot.RunSeries(mid, hard, n, 2002))
	printSeries("Easy vs Hard", bot.RunSeries(easy, hard, n, 3003))
}

func printSeries(title string, s bot.SeriesStats) {
	fmt.Printf("%s\n", title)
	fmt.Printf("  games=%d first_wins=%d second_wins=%d win_rate=%.3f CI[%.3f, %.3f]\n",
		s.Games, s.FirstWins, s.SecondWins, s.FirstWinRate, s.CILow, s.CIHigh)
	fmt.Printf("  avg_decision_ms first=%.1f second=%.1f\n\n", s.AvgFirstMs, s.AvgSecondMs)
}
