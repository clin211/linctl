// Package templates is the embedded template FS for lin v2.
//
// embed path is relative to this .go file's directory, so
// //go:embed all:project embeds everything under project/.
// //go:embed all:resource embeds everything under resource/.
package templates

import "embed"

//go:embed all:project all:resource
var FS embed.FS
