package ui

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"bg/internal/config"
	"bg/internal/game"
)

type snapJob struct {
	app *App
	out string
}

// boardSnap renders one prepared App per frame to a PNG. RunGame may only
// run once per process, so all jobs share a single game loop.
type boardSnap struct {
	jobs   []snapJob
	frames int
	err    error
	wrote  int
}

func (s *boardSnap) Update() error {
	s.frames++
	i := s.frames - 2 // first frame warms up the GPU context
	if i >= 0 && i < len(s.jobs) {
		img := ebiten.NewImage(screenW, screenH)
		s.jobs[i].app.Draw(img)
		f, err := os.Create(s.jobs[i].out)
		if err != nil {
			s.err = err
			return ebiten.Termination
		}
		s.err = png.Encode(f, img)
		_ = f.Close()
		s.wrote++
	}
	if i >= len(s.jobs) {
		return ebiten.Termination
	}
	return nil
}

func (s *boardSnap) Draw(screen *ebiten.Image) {
	if len(s.jobs) > 0 {
		s.jobs[0].app.Draw(screen)
	}
}

func (s *boardSnap) Layout(_, _ int) (int, int) { return screenW, screenH }

// TestWriteBoardSnapshots renders each theme to a PNG for visual comparison
// against the reference artwork. Set BG_SNAP to an output directory (or "1"
// for the repository root) and optionally BG_SNAP_SCENE=midgame.
func TestWriteBoardSnapshots(t *testing.T) {
	dir := os.Getenv("BG_SNAP")
	if dir == "" {
		t.Skip("set BG_SNAP=<dir> to write board snapshots")
	}
	if dir == "1" {
		dir = filepath.Join("..", "..")
	}
	scenario := os.Getenv("BG_SNAP_SCENE")
	var s boardSnap
	for _, id := range []ThemeID{ThemePersian, ThemeChinese, ThemeClassic} {
		a, err := New(config.Config{Theme: string(id), MatchTarget: 5})
		if err != nil {
			t.Fatal(err)
		}
		a.selected = 1
		a.startGame()
		switch scenario {
		case "midgame":
			var b game.Board
			b.Points[13], b.Points[8], b.Points[6], b.Points[4] = 4, 3, 5, 1
			b.Points[24], b.Points[12], b.Points[1] = -2, -4, -5
			b.Points[20] = -2
			b.Bar[0], b.Bar[1] = 1, 1
			b.Off[0], b.Off[1] = 1, 2
			a.board = b
			a.dice = [2]int{6, 3}
			r := game.NewRoll(a.board, game.White, 6, 3)
			a.roll = &r
			a.rolled = true
			a.showDice = true
			a.hasFrom = true
			a.selFrom = 13
		default:
			a.dice = [2]int{6, 3}
			r := game.NewRoll(a.board, game.White, 6, 3)
			a.roll = &r
			a.rolled = true
			a.showDice = true
		}
		out, err := filepath.Abs(filepath.Join(dir, "snap_"+string(id)+".png"))
		if err != nil {
			t.Fatal(err)
		}
		s.jobs = append(s.jobs, snapJob{app: a, out: out})
	}
	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetRunnableOnUnfocused(true)
	if err := ebiten.RunGame(&s); err != nil {
		t.Fatal(err)
	}
	if s.err != nil {
		t.Fatal(s.err)
	}
	if s.wrote != len(s.jobs) {
		t.Fatalf("wrote %d of %d snapshots", s.wrote, len(s.jobs))
	}
	for _, j := range s.jobs {
		if st, err := os.Stat(j.out); err != nil || st.Size() < 1000 {
			t.Fatalf("snapshot missing: %v", err)
		}
		t.Log("wrote", j.out)
	}
}
