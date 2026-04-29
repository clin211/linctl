{{- $app := (index .Project.Spec.Components 0).Name -}}
{{- $project := .Project.Metadata.Name -}}
{{- $registry := .Project.Spec.Defaults.Image.RegistryPrefix | default "ghcr.io/example" -}}
# CI/CD：main 分支推送时自动构建二进制 + Docker 镜像，并 SSH 部署到远端。
#
# 必备 GitHub Secret（Settings → Secrets and variables → Actions）：
#   DOCKER_USERNAME / DOCKER_PASSWORD —— 容器仓库账号（{{ $registry }}）
#   HOST                              —— 远端服务器地址（IP 或域名）
#   SSH_USER (可选)                   —— 远端登录用户，默认 root
#   SSH_PRIVATE_KEY                   —— 远端的 SSH 私钥（无 passphrase）
#
# 必备 Variables（可选，若想免改 yaml）：
#   IMAGE_REGISTRY  —— 默认沿用模板渲染期注入的 {{ $registry }}
#   IMAGE_REPO      —— 默认 {{ $project }}/{{ $app }}
#
# 提示：本文件使用的 ${{"{{"}} secrets.XXX {{"}}"}} / ${{"{{"}} env.YYY {{"}}"}} 是 GitHub Actions
# 内置的表达式语法（与 linctl 模板无关），由 GitHub 在执行 workflow 时求值。
name: Build & Deploy ({{ $app }})
on:
  push:
    branches:
      - "main"

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Compute version
        run: |
          echo "VERSION=$(git describe --tags --always --match='v*')" >> $GITHUB_ENV

      - name: Print version
        run: |
          echo "Building version ${{"{{"}} env.VERSION {{"}}"}}"

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.25"
          cache: true

      - name: Build binary
        working-directory: .
        run: |
          ls -l
          make build VERSION=${{"{{"}} env.VERSION {{"}}"}} BINS={{ $app }}

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to container registry
        # TODO: 如使用 GHCR，registry 改成 ghcr.io；阿里云为
        #   registry.cn-chengdu.aliyuncs.com
        run: |
          echo "Login to {{ $registry }}"
          REGISTRY_HOST="$(echo '{{ $registry }}' | cut -d'/' -f1)"
          docker login \
            --username=${{"{{"}} secrets.DOCKER_USERNAME {{"}}"}} \
            --password=${{"{{"}} secrets.DOCKER_PASSWORD {{"}}"}} \
            "$REGISTRY_HOST"

      - name: Build & push Docker image
        working-directory: .
        run: |
          IMAGE_TAG="{{ $registry }}/{{ $app }}:${{"{{"}} env.VERSION {{"}}"}}"
          echo "Building Docker image: $IMAGE_TAG"
          docker build . \
            --file Dockerfile \
            --build-arg OS=linux \
            --build-arg ARCH=amd64 \
            --build-arg BIN={{ $app }} \
            --tag "$IMAGE_TAG"

          echo "Pushing $IMAGE_TAG ..."
          docker push "$IMAGE_TAG"

      - name: Deploy to remote server
        # TODO: 远端需要预先存在 ${REMOTE_DIR}/setup.sh 用于 docker-compose pull/up
        uses: appleboy/ssh-action@v1.0.3
        with:
          host: ${{"{{"}} secrets.HOST {{"}}"}}
          username: ${{"{{"}} secrets.SSH_USER || 'root' {{"}}"}}
          key: ${{"{{"}} secrets.SSH_PRIVATE_KEY {{"}}"}}
          port: 22
          script: |
            VERSION='${{"{{"}} env.VERSION {{"}}"}}'
            echo "Deploying ${VERSION} to $(hostname)"
            REMOTE_DIR="/home/project/{{ $project }}"
            cd "$REMOTE_DIR"
            sh setup.sh '${{"{{"}} secrets.DOCKER_USERNAME {{"}}"}}' '${{"{{"}} secrets.DOCKER_PASSWORD {{"}}"}}' "${VERSION}"
