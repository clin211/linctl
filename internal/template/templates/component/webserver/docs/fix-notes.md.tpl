# 错误处理重构修复笔记

## 需要修复的文件和位置

以下文件中使用了已弃用的 `WithMessage` 方法，需要修复：

### 1. internal/{{ .Component.Name }}/pkg/validation/user.go

将 `WithMessage` 替换为 `WithDetails`：

```go
// 原代码
errno.ErrInvalidArgument.WithMessage("用户名格式不正确")

// 修复后
errno.ErrInvalidArgument.WithDetails("用户名格式不正确")
```

### 2. internal/{{ .Component.Name }}/pkg/validation/validation.go

同样将 `WithMessage` 替换为 `WithDetails`：

```go
// 原代码
errno.ErrInvalidArgument.WithMessage("参数验证失败")

// 修复后
errno.ErrInvalidArgument.WithDetails("参数验证失败")
```

### 3. internal/pkg/middleware/gin/authn.go

```go
// 原代码
errno.ErrTokenInvalid.WithMessage("Token格式无效")
errno.ErrUnauthenticated.WithMessage("用户未登录")

// 修复后
errno.ErrTokenInvalid.WithDetails("Token格式无效")
errno.ErrUnauthenticated.WithDetails("用户未登录")
```

### 4. internal/pkg/middleware/gin/authz.go

```go
// 原代码
errno.ErrPermissionDenied.WithMessage("权限不足")

// 修复后
errno.ErrPermissionDenied.WithDetails("权限不足")
```

### 5. internal/{{ .Component.Name }}/biz/v1/user/create.go

```go
// 原代码
errno.ErrAddRole.WithMessage("添加用户角色失败")

// 修复后
errno.ErrAddRole.WithDetails("添加用户角色失败")
```

### 6. internal/{{ .Component.Name }}/biz/v1/user/delete.go

```go
// 原代码
errno.ErrRemoveRole.WithMessage("删除用户角色失败")

// 修复后
errno.ErrRemoveRole.WithDetails("删除用户角色失败")
```

## 批量修复脚本

可以使用以下 sed 命令进行批量替换：

```bash
find . -name "*.go" -type f -exec sed -i 's/\.WithMessage(/.WithDetails(/g' {} \;
```

## 验证修复

修复后运行以下命令验证：

```bash
go build ./...
go test ./pkg/errorsx/...
```

## 其他注意事项

1. **语义差异**：`WithMessage` 是替换整个消息，而 `WithDetails` 是添加详细信息。在新体系中，主要消息保持不变，详细信息存储在 `Details` 字段中。

2. **前端处理**：前端需要适配新的响应格式，错误详情现在在 `error.details` 字段中。

3. **监控和日志**：错误详情会被包含在日志中，便于问题排查。

4. **向后兼容**：现有的 API 调用方仍然能够正常工作，只需要逐步迁移到新的错误处理方式。
