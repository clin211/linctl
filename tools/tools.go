//go:build tools
// +build tools

// Package tools 用于 go.mod 跟踪开发工具的版本（即使它们不在生产代码中导入）。
//
// 真正的安装通过 Makefile `make tools` 完成，这里仅为可复现的版本声明。
package tools

import (
	_ "go.uber.org/mock/mockgen"
	_ "mvdan.cc/gofumpt"
)
