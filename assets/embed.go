// Package assets embeds fonts and board artwork for the desktop client.
package assets

import _ "embed"

// Font is Liberation Sans (SIL Open Font License). See fonts/OFL.txt.
//
//go:embed fonts/LiberationSans-Regular.ttf
var Font []byte

// PersianOrnateBoard is the reference artwork for the Persian theme
// (1536×1024). The UI cuts frame, points, medallion, and checker sprites
// from it at load time; it is never shown as a single static picture.
//
//go:embed board/persian_ornate.jpg
var PersianOrnateBoard []byte

// ChineseBoard is the reference artwork for the Chinese theme (1024×559,
// a perspective render). The UI un-warps the board plane and cuts frame,
// bar, point, and checker sprites from the result.
//
//go:embed board/chinese1.jpg
var ChineseBoard []byte
