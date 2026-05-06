// Package templates 是 lin v2 的内嵌模板 FS。
//
// embed 路径相对于本 .go 文件所在目录，因此：
// //go:embed all:project 会嵌入 project/ 下的所有内容。
// //go:embed all:resource 会嵌入 resource/ 下的所有内容。
package templates

import "embed"

//go:embed all:project all:resource
var FS embed.FS
