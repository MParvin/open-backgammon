// Package assets embeds fonts and board artwork for the desktop client.
package assets

import _ "embed"

// Font is Liberation Sans (SIL Open Font License). See fonts/OFL.txt.
//
//go:embed fonts/LiberationSans-Regular.ttf
var Font []byte

// PersianOrnateBoard is the cleaned plate from "Ornate Persian Backgammon Board.png"
// (checkers, dice, and score glyphs removed for gameplay overlay).
//
//go:embed board/persian_ornate.jpg
var PersianOrnateBoard []byte
