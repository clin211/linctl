# 程序与插件二进制
*.exe
*.exe~
*.dll
*.so
*.dylib

# 通过 `go test -c` 构建出的测试二进制
*.test

# go coverage 工具的输出（尤其是配合 LCov 使用时）
*.out

# 依赖目录（推荐使用 module cache 替代）
vendor/

# Go workspace 文件
go.work
go.work.sum

# 构建产物
_output/

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# macOS
.DS_Store

# 环境变量文件
.env
.env.*
!.env.example

# 含密钥的配置
configs/*.local.yaml
configs/*.secret.yaml

# 日志文件
*.log
