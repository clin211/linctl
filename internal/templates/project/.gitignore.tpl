# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with `go test -c`
*.test

# Output of the go coverage tool, specifically when used with LCov
*.out

# Dependency directories (use module caches instead)
vendor/

# Go workspace file
go.work
go.work.sum

# Build output
_output/

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# macOS
.DS_Store

# lin backup and last-run metadata
.lin/.backup/
.lin/.last-run.json

# Environment files
.env
.env.*
!.env.example

# Config with secrets
configs/*.local.yaml
configs/*.secret.yaml

# Log files
*.log
