// Command backgammon is the Linux desktop backgammon client.
package main

import (
	"errors"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"bg/internal/config"
	"bg/internal/ui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	app, err := ui.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowTitle("Backgammon")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetRunnableOnUnfocused(true)
	if err := ebiten.RunGame(app); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
}
