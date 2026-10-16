package game

import "testing"

func TestClassifySingleGammonBackgammon(t *testing.T) {
	var single Board
	single.Off[colorIndex(White)] = 15
	single.Off[colorIndex(Black)] = 1
	single.Points[24] = -1
	if Classify(single, White) != Single {
		t.Fatal("single")
	}

	var gammon Board
	gammon.Off[colorIndex(White)] = 15
	gammon.Points[12] = -5
	gammon.Points[17] = -5
	gammon.Points[19] = -5
	if Classify(gammon, White) != Gammon {
		t.Fatalf("got %v", Classify(gammon, White))
	}

	var bg Board
	bg.Off[colorIndex(White)] = 15
	bg.Bar[colorIndex(Black)] = 1
	bg.Points[19] = -14
	if Classify(bg, White) != Backgammon {
		t.Fatalf("got %v", Classify(bg, White))
	}

	var bgHome Board
	bgHome.Off[colorIndex(White)] = 15
	bgHome.Points[3] = -2
	bgHome.Points[19] = -13
	if Classify(bgHome, White) != Backgammon {
		t.Fatalf("home bg %v", Classify(bgHome, White))
	}
}

func TestMatchCrawfordAndFinish(t *testing.T) {
	m := NewMatch(5)
	m = m.ApplyGame(White, Single, 1) // 1-0
	m = m.ApplyGame(White, Single, 1) // 2-0
	m = m.ApplyGame(White, Single, 1) // 3-0
	m = m.ApplyGame(White, Single, 1) // 4-0 -> Crawford next
	if m.Finished() || !m.Crawford || m.GameNum != 5 {
		t.Fatalf("crawford setup %+v", m)
	}
	m = m.ApplyGame(Black, Single, 1) // Crawford game, 4-1
	if m.Crawford || !m.PostCrawford {
		t.Fatalf("after crawford %+v", m)
	}
	m = m.ApplyGame(White, Single, 1) // 5-1 match over
	if !m.Finished() || *m.Winner() != White {
		t.Fatalf("finish %+v", m)
	}
}

func TestCubeOfferTake(t *testing.T) {
	c := NewCube()
	white := White
	if !CanOffer(c, White, false, false, false) {
		t.Fatal("center open")
	}
	c = c.Take(Black)
	if c.Value != 2 || c.Owner == nil || *c.Owner != Black {
		t.Fatalf("%+v", c)
	}
	if CanOffer(c, White, false, false, false) {
		t.Fatal("white does not own cube")
	}
	if !CanOffer(c, Black, false, false, false) {
		t.Fatal("black owns cube")
	}
	if CanOffer(c, Black, true, false, false) {
		t.Fatal("crawford blocks")
	}
	_ = white
}

func TestPoints(t *testing.T) {
	if Points(Gammon, 2) != 4 || Points(Backgammon, 4) != 12 {
		t.Fatal(Points(Gammon, 2), Points(Backgammon, 4))
	}
}
