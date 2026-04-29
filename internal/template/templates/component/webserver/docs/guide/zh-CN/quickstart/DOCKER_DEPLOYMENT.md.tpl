# Docker 部署指南

本文档说明如何在不同场景下部署 `{{ .Component.Name }}` 服务。

## 目录

- [问题背景](#问题背景)
- [解决方案](#解决方案)
- [本地开发环境部署](#本地开发环境部署)
- [生产环境部署](#生产环境部署)
- [常见问题](#常见问题)

## 问题背景

在 Docker 容器中，`127.0.0.1` 或 `localhost` 指向的是**容器自己的网络命名空间**，而不是宿主机。因此，当应用尝试连接宿主机上运行的数据库、Redis 等服务时会失败：

```
Error: dial tcp 127.0.0.1:54321: connect: connection refused
```

## 解决方案

根据不同的部署场景，我们提供了多种配置方案：

### 1. 数据库在同一宿主机上（开发/测试环境）

使用 `host.docker.internal` 访问宿主机服务。

**配置文件示例**：`configs/{{ .Component.Name }}.docker.yaml`

```yaml
postgresql:
  addr: host.docker.internal:54321  # 访问宿主机的 PostgreSQL
```

**docker-compose.yml 配置**：

```yaml
services:
  {{ .Component.Name }}:
    extra_hosts:
      - "host.docker.internal:host-gateway"  # Linux 必需
```

### 2. 数据库在其他服务器上（生产环境）

直接使用数据库服务器的 IP 地址和端口。

**配置文件示例**：`configs/{{ .Component.Name }}.prod.yaml`

```yaml
postgresql:
  addr: 192.168.1.100:5432  # 数据库服务器的实际 IP
```

### 3. 数据库在同一 Docker 网络中

使用 Docker 服务名称进行通信。

**配置文件示例**：

```yaml
postgresql:
  addr: postgres-container:5432  # 使用容器名或服务名
```

**docker-compose.yml 配置**：

```yaml
services:
  {{ .Component.Name }}:
    networks:
      - app-network
  postgres:
    networks:
      - app-network
networks:
  app-network:
    driver: bridge
```

## 本地开发环境部署

### 步骤 1: 构建应用

```bash
# 构建二进制文件
make build BINS={{ .Component.Name }}

# 构建 Docker 镜像
make image PLATFORM=linux_amd64 VERSION=v0.0.5-alpha IMAGES={{ .Component.Name }}
```

### 步骤 2: 启动依赖服务

```bash
# 启动 PostgreSQL、Redis、OTEL Collector
docker compose -f docker-compose.env.yml up -d
```

### 步骤 3: 启动应用服务

```bash
cd build/docker/{{ .Component.Name }}

# 启动应用
docker compose up -d

# 查看日志
docker compose logs -f

# 测试健康检查
curl localhost:5556/healthz
```

### 步骤 4: 停止服务

```bash
# 停止应用
docker compose down

# 停止所有服务（包括依赖）
cd /path/to/project/root
docker compose -f docker-compose.env.yml down
```

## 生产环境部署

### 场景 A: 数据库在独立服务器上

适用于微服务架构，数据库、Redis、监控服务分布在不同服务器。

#### 1. 准备配置文件

```bash
# 复制配置模板
cp configs/{{ .Component.Name }}.prod.yaml.example configs/{{ .Component.Name }}.prod.yaml

# 编辑配置文件
vim configs/{{ .Component.Name }}.prod.yaml
```

修改以下配置：

```yaml
# JWT 密钥（必须修改）
jwt-key: YOUR_SECURE_RANDOM_STRING_AT_LEAST_32_CHARS

# 数据库地址（替换为实际 IP）
postgresql:
  addr: 192.168.1.100:5432
  password: PRODUCTION_PASSWORD

# Redis 地址（替换为实际 IP）
redis:
  addr: 192.168.1.101:6379
  password: REDIS_PASSWORD

# OTEL Collector 地址（替换为实际 IP）
otel:
  endpoint: 192.168.1.102:4327
```

#### 2. 准备部署目录

```bash
cd build/docker/{{ .Component.Name }}

# docker-compose.yml 使用相对路径，自动从项目根目录加载配置
# 相对路径：../../../configs/{{ .Component.Name }}.prod.yaml
# 无需修改配置文件路径
```

如果需要使用其他位置的配置文件，可以修改 `docker-compose.prod.yml`：

```yaml
volumes:
  # 使用相对路径（推荐）
  - ../../../configs/{{ .Component.Name }}.prod.yaml:/app/configs/{{ .Component.Name }}.yaml:ro
  # 或使用绝对路径
  # - /path/to/configs/{{ .Component.Name }}.prod.yaml:/app/configs/{{ .Component.Name }}.yaml:ro
```

#### 3. 部署应用

```bash
# 设置版本号并启动
VERSION=v0.0.5-alpha docker compose -f docker-compose.prod.yml up -d

# 查看日志
docker compose -f docker-compose.prod.yml logs -f

# 检查健康状态
curl http://localhost:5556/healthz
```

### 场景 B: 所有服务在同一宿主机

适用于单机部署场景。

#### 1. 使用开发环境配置

配置文件使用 `host.docker.internal`：

```yaml
postgresql:
  addr: host.docker.internal:54321
redis:
  addr: host.docker.internal:56379
otel:
  endpoint: host.docker.internal:4327
```

#### 2. 确保 docker-compose 包含 extra_hosts

```yaml
services:
  {{ .Component.Name }}:
    extra_hosts:
      - "host.docker.internal:host-gateway"
```

#### 3. 部署

```bash
# 先启动依赖服务
docker compose -f docker-compose.env.yml up -d

# 再启动应用
cd build/docker/{{ .Component.Name }}
docker compose up -d
```

### 场景 C: 使用 Docker 网络（推荐用于容器化部署）

所有服务都运行在容器中，使用同一个 Docker 网络。

#### 1. 创建统一的 docker-compose.yml

```yaml
services:
  postgres:
    image: postgres:16-alpine
    container_name: postgres
    networks:
      - app-network
    # ... 其他配置

  redis:
    image: redis:7-alpine
    container_name: redis
    networks:
      - app-network
    # ... 其他配置

  {{ .Component.Name }}:
    image: {{ .Project.Metadata.Name }}/{{ .Component.Name }}:v0.0.5-alpha
    depends_on:
      - postgres
      - redis
    networks:
      - app-network
    # ... 其他配置

networks:
  app-network:
    driver: bridge
```

#### 2. 配置文件使用服务名

```yaml
postgresql:
  addr: postgres:5432
redis:
  addr: redis:6379
```

## 常见问题

### Q1: 容器启动后立即退出

**原因**：无法连接数据库或其他依赖服务。

**解决方法**：

```bash
# 查看日志
docker logs {{ .Component.Name }}

# 检查网络配置
docker inspect {{ .Component.Name }} | grep -A 20 "Networks"

# 测试连接
docker exec -it {{ .Component.Name }} /bin/sh  # 注意：scratch 镜像无 shell
```

### Q2: Linux 服务器上 host.docker.internal 不可用

**原因**：Linux 默认不支持 `host.docker.internal`。

**解决方法**：在 docker-compose.yml 中添加：

```yaml
extra_hosts:
  - "host.docker.internal:host-gateway"
```

### Q3: Connection refused 错误

**原因**：配置文件中的地址不正确。

**诊断步骤**：

1. **确认服务端口映射**：

   ```bash
   docker ps  # 查看端口映射
   ```

2. **测试宿主机服务**：

   ```bash
   # 从宿主机测试
   telnet localhost 54321  # PostgreSQL
   telnet localhost 56379  # Redis
   ```

3. **容器内测试（如果有 shell）**：

   ```bash
   docker exec -it {{ .Component.Name }} ping host.docker.internal
   ```

### Q4: 如何查看应用日志

```bash
# 实时查看日志
docker logs -f {{ .Component.Name }}

# 查看最近 100 行
docker logs --tail 100 {{ .Component.Name }}

# 查看特定时间范围
docker logs --since 30m {{ .Component.Name }}
```

### Q5: 如何更新应用版本

```bash
# 1. 构建新版本镜像
make image PLATFORM=linux_amd64 VERSION=v0.0.6-alpha IMAGES={{ .Component.Name }}

# 2. 更新 docker-compose.yml 中的版本号
vim build/docker/{{ .Component.Name }}/docker-compose.yml
# 修改 image: {{ .Project.Metadata.Name }}/{{ .Component.Name }}:v0.0.6-alpha

# 3. 重启服务
cd build/docker/{{ .Component.Name }}
docker compose down
docker compose up -d

# 4. 验证
curl localhost:5556/healthz
```

## 监控和维护

### 健康检查

```bash
# HTTP 健康检查
curl http://localhost:5556/healthz

# 检查容器健康状态
docker inspect {{ .Component.Name }} | grep -A 10 "Health"
```

### 性能监控

```bash
# 查看资源使用
docker stats {{ .Component.Name }}

# 查看容器详细信息
docker inspect {{ .Component.Name }}
```

### 备份和恢复

```bash
# 导出镜像
docker save {{ .Project.Metadata.Name }}/{{ .Component.Name }}:v0.0.5-alpha -o {{ .Component.Name }}-v0.0.5.tar

# 导入镜像
docker load -i {{ .Component.Name }}-v0.0.5.tar
```

## 安全建议

1. **配置文件权限**：

   ```bash
   chmod 600 configs/{{ .Component.Name }}.prod.yaml
   ```

2. **使用环境变量管理敏感信息**（可选）：

   ```yaml
   environment:
     - DB_PASSWORD=${DB_PASSWORD}
     - REDIS_PASSWORD=${REDIS_PASSWORD}
   ```

3. **定期更新镜像**：

   ```bash
   # 更新基础镜像
   docker pull golang:1.25.3
   docker pull gcr.io/distroless/base-debian12:nonroot
   
   # 重新构建
   make image PLATFORM=linux_amd64 VERSION=vX.X.X IMAGES={{ .Component.Name }}
   ```

4. **网络隔离**：生产环境使用专用 Docker 网络，避免容器直接暴露。

## 相关文件

- `configs/{{ .Component.Name }}.yaml` - 本地开发配置
- `configs/{{ .Component.Name }}.docker.yaml` - Docker 开发环境配置
- `configs/{{ .Component.Name }}.prod.yaml.example` - 生产环境配置模板
- `build/docker/{{ .Component.Name }}/docker-compose.yml` - 开发环境 compose 文件
- `build/docker/{{ .Component.Name }}/docker-compose.prod.yml` - 生产环境 compose 文件
- `docker-compose.env.yml` - 依赖服务 compose 文件

## 总结

选择合适的部署方案：

| 场景 | 推荐方案 | 配置地址格式 |
|------|---------|-------------|
| 本地开发 | host.docker.internal | `host.docker.internal:PORT` |
| 单机部署（容器+宿主机服务） | host.docker.internal | `host.docker.internal:PORT` |
| 单机部署（全容器化） | Docker 网络 | `service-name:PORT` |
| 微服务/分布式部署 | IP 地址 | `192.168.x.x:PORT` |
| 云环境 | 服务发现/DNS | `service.namespace:PORT` |

如有问题，请查看日志：`docker logs {{ .Component.Name }}`
