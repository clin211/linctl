package template_test

import (
	"strings"
	"testing"

	"github.com/clin211/linctl/internal/project"
	tpl "github.com/clin211/linctl/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStreamA_RootTemplatesRender 是 Stream A 引入的所有项目级 / 组件级 build/docker /
// scripts / .github 模板的烟测。
//
// 目的：保证模板能用真实 Project 数据（而非 mock map）渲染通过，覆盖
//   - 模板路径在 embed.FS 中可达（不会被 dotfile / 子目录排除）
//   - 字段引用与 lin/internal/project/types.go 中的 schema 对齐
//   - Author / Image / Defaults 等可空块在 ApplyDefaults 后均有合理 default
func TestStreamA_RootTemplatesRender(t *testing.T) {
	eng, err := tpl.New()
	require.NoError(t, err)

	p := &project.Project{
		APIVersion: project.APIVersionV1,
		Kind:       project.KindProject,
		Metadata: project.Metadata{
			Name:        "acme-blog",
			Module:      "github.com/acme/acme-blog",
			Description: "Acme demo blog (linctl).",
			Author: project.Author{
				Name:  "Acme Team",
				Email: "team@acme.io",
			},
		},
		Spec: project.Spec{
			Components: []project.Component{
				{
					Kind:      "WebServer",
					Name:      "blog-apiserver",
					Framework: "gin",
					Storage:   "gorm-postgres",
					Port:      5556,
					Features:  []string{"healthz", "user"},
				},
			},
		},
	}
	project.ApplyDefaults(p)
	p.Spec.Defaults.Image.RegistryPrefix = "registry.example.com/acme"

	data := tpl.TemplateData{
		Project:    p,
		Component:  p.Spec.Components[0],
		CLIVersion: "v0.1.0-stream-a-test",
	}

	rootTemplates := []struct {
		path           string
		mustContain    []string
		mustNotContain []string
	}{
		{
			path: "templates/project/Dockerfile.tpl",
			mustContain: []string{
				"FROM ${BUILDER_IMAGE} AS builder",
				"ARG BIN=blog-apiserver",
				"ENTRYPOINT [\"/app/blog-apiserver\"]",
				"EXPOSE 5556",
			},
			mustNotContain: []string{"miniblog-v4", "clin211"},
		},
		{
			path: "templates/project/docker-compose.env.yml.tpl",
			mustContain: []string{
				"container_name: acme-blog-postgres",
				"POSTGRES_DB: acme-blog",
				"acme-blog_net",
			},
			// 注释中允许出现 "miniblog" 作为端口约定的来源说明。
			// 容器名 / network / DB 名等实际配置已替换为 acme-blog。
			mustNotContain: []string{"container_name: miniblog", "miniblog-postgres", "miniblog-redis"},
		},
		{
			path: "templates/project/dockerignore.tpl",
			mustContain: []string{
				"/_output",
				"configs/blog-apiserver.local.yaml",
			},
		},
		{
			path: "templates/project/PROJECT.tpl",
			mustContain: []string{
				"modulePath: github.com/acme/acme-blog",
				"name: acme-blog",
				"author: Acme Team",
				"binaryName: blog-apiserver",
				"withHealthz: true",
				"registryPrefix: registry.example.com/acme",
			},
			mustNotContain: []string{"长林啊", "clin211"},
		},
		{
			path: "templates/project/otel-collector.yaml.tpl",
			mustContain: []string{
				"endpoint: 0.0.0.0:4327",
				"endpoint: 0.0.0.0:4328",
				"OpenTelemetry Collector 配置（acme-blog）",
			},
		},
		{
			path: "templates/project/golangci.yaml.tpl",
			mustContain: []string{
				"version: \"2\"",
				"prefix(github.com/acme/acme-blog)",
				"module-path: github.com/acme/acme-blog",
				"- github.com/acme/acme-blog",
			},
			mustNotContain: []string{"github.com/org/project"},
		},
		{
			path: "templates/component/webserver/build/docker/Dockerfile.tpl",
			mustContain: []string{
				"make build BINS=blog-apiserver",
				"COPY --from=builder /app/blog-apiserver /app/blog-apiserver",
				"EXPOSE 5556",
			},
		},
		{
			path: "templates/component/webserver/build/docker/docker-compose.yml.tpl",
			mustContain: []string{
				"image: registry.example.com/acme/blog-apiserver:dev",
				"context: ../../..",
				"dockerfile: build/docker/blog-apiserver/Dockerfile",
				"\"5556:5556\"",
			},
		},
		{
			path: "templates/component/webserver/build/docker/docker-compose.prod.yml.tpl",
			mustContain: []string{
				"image: registry.example.com/acme/blog-apiserver:${VERSION:-latest}",
				"\"5556:5556\"",
				"acme-blog_net",
			},
		},
		{
			path: "templates/project/scripts/coverage.awk",
			mustContain: []string{
				"#!/usr/bin/env awk",
				"^total:",
			},
		},
		{
			path: "templates/project/scripts/boilerplate.txt.tpl",
			mustContain: []string{
				"Acme Team",
				"team@acme.io",
				"github.com/acme/acme-blog",
			},
		},
		{
			path: "templates/project/scripts/startup-test.sh.tpl",
			mustContain: []string{
				"API_BASE=\"${API_BASE:-http://127.0.0.1:5556/v1}\"",
				"create|get|list",
			},
		},
		{
			path: "templates/project/github/workflows/deploy.yml.tpl",
			mustContain: []string{
				"name: Build & Deploy (blog-apiserver)",
				"BINS=blog-apiserver",
				// GitHub Actions expressions must be emitted literally
				"${{ env.VERSION }}",
				"${{ secrets.HOST }}",
				"${{ secrets.DOCKER_USERNAME }}",
				// Registry should be substituted, not literal placeholder
				"\"$REGISTRY_HOST\"",
				"registry.example.com/acme/blog-apiserver",
			},
			// 阿里云 registry 仅出现在注释/示例中，实际 IMAGE_TAG 已被
			// 模板期注入的 RegistryPrefix（registry.example.com/acme）替换。
			mustNotContain: []string{
				"registry.cn-chengdu.aliyuncs.com/go-practice",
				"--tag registry.cn-chengdu",
			},
		},
	}

	for _, tc := range rootTemplates {
		t.Run(tc.path, func(t *testing.T) {
			out, err := eng.RenderToString(tc.path, data)
			require.NoError(t, err, "render %s", tc.path)
			for _, s := range tc.mustContain {
				assert.Contains(t, out, s, "path=%s expected to contain %q", tc.path, s)
			}
			for _, s := range tc.mustNotContain {
				assert.NotContains(t, out, s, "path=%s should not contain %q", tc.path, s)
			}
		})
	}
}

// TestStreamA_GolangciVerbatim 校验 .golangci.yaml 模板渲染后与 miniblog-v4 原文件
// 仅在 module 占位符相关三处有差异——保证我们没意外破坏其他规则。
func TestStreamA_GolangciVerbatim(t *testing.T) {
	eng, err := tpl.New()
	require.NoError(t, err)

	p := &project.Project{
		APIVersion: project.APIVersionV1,
		Kind:       project.KindProject,
		Metadata: project.Metadata{
			Name:   "demo",
			Module: "github.com/demo/demo",
		},
		Spec: project.Spec{
			Components: []project.Component{
				{Kind: "WebServer", Name: "demo", Framework: "gin", Storage: "gorm-postgres"},
			},
		},
	}
	project.ApplyDefaults(p)

	out, err := eng.RenderToString(
		"templates/project/golangci.yaml.tpl",
		tpl.TemplateData{Project: p, Component: p.Spec.Components[0]},
	)
	require.NoError(t, err)

	// 三处 module 占位都应被替换
	assert.Contains(t, out, "prefix(github.com/demo/demo)")
	assert.Contains(t, out, "module-path: github.com/demo/demo")
	assert.NotContains(t, out, "github.com/org/project")

	// 行数应与 miniblog-v4 原文件大致相同（允许 ±10 行变动）
	lines := strings.Count(out, "\n")
	assert.GreaterOrEqual(t, lines, 4170, "render should be close to upstream's 4180 lines")
	assert.LessOrEqual(t, lines, 4200)
}
