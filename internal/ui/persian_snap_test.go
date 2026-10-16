package ui

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"bg/internal/config"
)

type persianSnap struct {
	*App
	frames int
	out    string
	done   bool
	err    error
}

func (s *persianSnap) Update() error {
	s.frames++
	if s.frames == 2 && !s.done && s.out != "" {
		img := ebiten.NewImage(screenW, screenH)
		s.App.Draw(img)
		f, err := os.Create(s.out)
		if err != nil {
			s.err = err
		} else {
			s.err = png.Encode(f, img)
			_ = f.Close()
		}
		s.done = true
	}
	if s.frames > 3 {
		return ebiten.Termination
	}
	return nil
}

func (s *persianSnap) Draw(screen *ebiten.Image) {
	s.App.Draw(screen)
}

func TestWritePersianBoardSnapshot(t *testing.T) {
	if os.Getenv("BG_SNAP") == "" {
		t.Skip("set BG_SNAP=1 to write persian_snap.png")
	}
	a, err := New(config.Config{Theme: "persian", MatchTarget: 5})
	if err != nil {
		t.Fatal(err)
	}
	a.selected = 1
	a.startGame()
	a.dice = [2]int{6, 3}
	a.showDice = true
	a.rolled = true
	out, err := filepath.Abs(filepath.Join("..", "..", "persian_snap.png"))
	if err != nil {
		t.Fatal(err)
	}
	s := &persianSnap{App: a, out: out}
	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetRunnableOnUnfocused(true)
	if err := ebiten.RunGame(s); err != nil {
		t.Fatal(err)
	}
	if s.err != nil {
		t.Fatal(s.err)
	}
	if !s.done {
		t.Fatal("Draw never wrote the snapshot")
	}
	if st, err := os.Stat(out); err != nil || st.Size() < 1000 {
		t.Fatalf("snapshot missing: %v", err)
	}
	t.Log("wrote", out)
}
