{{- $app := (index .Project.Spec.Components 0).Name -}}
# 缩小 docker build 上下文，加速构建并避免泄漏开发文件。
/.vscode
/.idea
/data
/_output
/dist
/.git
README.md
docker-compose.env.yml
docker-compose*.yml
configs/{{ $app }}.local.yaml
*.test
*.out
.env
.env.*
