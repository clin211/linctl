# {{.ProjectName | Title}}

## 项目概览

{{.ProjectName | Title}} 是一个采用分层架构（handler / biz / store，受 DDD 启发）的 Go 后端服务。

## 技术栈

- **Web 框架**：{{.Framework}}
- **存储层**：{{.Storage}}
- **共享库**：[linhub](https://github.com/clin211/linhub)（`github.com/clin211/linhub`）—— 提供 db、log、errx、core、store、options 等基础能力
- **语言**：Go {{.GoVersion}}

## 快速开始

```bash
# 安装依赖
make deps

# 构建
make build

# 运行
./_output/{{.AppName}} -c configs/{{.AppName}}.yaml
```

## 项目结构

```
{{.ProjectName}}/
├── cmd/{{.AppName}}/          # 应用入口
├── internal/{{.AppName}}/     # 应用内部包
│   ├── handler/               # HTTP handler
│   ├── biz/                   # 业务逻辑层
│   └── store/                 # 数据访问层
├── internal/pkg/              # 内部共享工具
├── pkg/                       # 公共包
└── configs/                   # 配置文件
```

## 配置

编辑 `configs/{{.AppName}}.yaml` 即可调整服务配置。

## 作者

{{.Author}} <{{.Email}}>
