package common

import (
	"embed"
	"encoding/base64"
	"strings"
)

// LOGORAW contains the raw SVG content of the NoLets logo.
const LOGORAW = `
<svg  id="noletLogo" xmlns="http://www.w3.org/2000/svg" version="1.1" viewBox="0 0 500 500" fill="currentColor">
  <polygon class="cls-1" points="99.8 457.6 67 493.4 20.4 482.3 6.5 435.5 39.2 399.6 85.9 410.8 99.8 457.6"/>
  <path class="cls-1" d="M493.5,54v343.4l-48.1,49.2h-.1c0,0-44.4-70-44.4-70v-209.7L133.2,440.6c0,.1-.2.1-.2.1,0,0-.4,0-.4-.4v-132.9l201.6-206.2H107.5l-58.1-28.5L114,6.6h333.2c.4,0,.8,0,1.2,0,25,.6,45.1,21.6,45.1,47.3Z"/>
  <path class="cls-1" d="M493.5,57v-3c0-25.7-20.1-46.6-45.1-47.3,25,.5,45.2,21.5,45.2,47.3s0,2,0,3Z"/>
</svg>
`

// StaticFS embeds the static files served by the application.
var StaticFS *embed.FS

// LogoSvgImage generates a base64-encoded SVG image or raw SVG string with the specified color.
func LogoSvgImage(color string, svg bool) string {
	color1 := "#ff0000"
	if color != "" {
		color1 = "#" + color
	}
	logosvg := strings.Replace(LOGORAW, "currentColor", color1, 1)
	if svg {
		return logosvg
	}
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(logosvg))
}
