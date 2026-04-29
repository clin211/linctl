# 跨层 i18n 使用指南

## 问题背景

原始的 i18n 中间件设计只适用于 HTTP Handler 层，因为在 Repository、Service 等业务层中没有 `*gin.Context`。这在分层架构中造成了限制。

## 解决方案

通过 Go 的 `context.Context` 来传递 i18n 实例，实现跨层的国际化支持。

## 架构设计

### 1. 中间件层 (Middleware)

```go
// middleware/i18n.go
func I18nMiddleware(i18nInstance *miniblogI18n.I18n) gin.HandlerFunc {
    return func(c *gin.Context) {
        // ... 语言检测逻辑 ...

        // 将 i18n 实例注入到 request.Context 中
        c.Request = c.Request.WithContext(
            miniblogI18n.WithContext(c.Request.Context(), localizer)
        )

        c.Next()
    }
}
```

### 2. Handler 层

```go
// handlers/user.go
func (h *UserHandler) GetUser(c *gin.Context) {
    // 方法1: 通过 Gin Context 获取 (适用于当前层)
    i18nInstance := middleware.GetI18n(c)

    // 方法2: 通过 context.Context 传递 (适用于后续层)
    // c.Request.Context() 已经包含 i18n 实例
    user, err := h.userService.GetUser(c.Request.Context(), userID)

    // 使用翻译
    message := i18nInstance.T("user.login_success")
}
```

### 3. Service 层

```go
// service/user.go
func (s *UserService) GetUser(ctx context.Context, userID int) (*User, error) {
    // 从 context 获取 i18n 实例
    i18nInstance := i18n.FromContext(ctx)

    // 调用 Repository 层，传递 context
    user, err := s.userRepo.FindUser(ctx, userID)
    if err != nil {
        // 在 Service 层处理错误翻译
        return nil, fmt.Errorf(i18nInstance.T("common.error") + ": " + err.Error())
    }

    return user, nil
}
```

### 4. Repository 层

```go
// repository/user.go
func (r *UserRepository) FindUser(ctx context.Context, userID int) (*User, error) {
    // 从 context 获取 i18n 实例
    i18nInstance := i18n.FromContext(ctx)

    // 回退方案：如果 context 中没有，使用实例默认 i18n
    if i18nInstance == nil {
        i18nInstance = r.i18n
    }

    if userID <= 0 {
        // 使用 i18n 翻译错误消息
        return nil, i18nInstance.E("user.user_not_found")
    }

    // ... 业务逻辑 ...
}
```

## Context 传递的关键点

### 1. 自动注入

中间件会自动将 i18n 实例注入到 `request.Context` 中：

```go
c.Request = c.Request.WithContext(miniblogI18n.WithContext(c.Request.Context(), localizer))
```

### 2. 获取方式

在任何层中都可以通过以下方式获取 i18n 实例：

```go
i18nInstance := i18n.FromContext(ctx)
```

### 3. 回退机制

为了保证代码的健壮性，建议在每层都提供回退机制：

```go
i18nInstance := i18n.FromContext(ctx)
if i18nInstance == nil {
    i18nInstance = r.defaultI18n // 实例的默认 i18n
}
```

## 测试示例

### 1. 启动服务器

```bash
cd examples/gin-i18n
go run main.go
```

### 2. 测试跨层 i18n

#### 获取用户信息 (Handler -> Service -> Repository)

```bash
# 默认英语
curl http://localhost:8080/api/v1/users/123

# 切换到中文
curl "http://localhost:8080/api/v1/users/123?lang=zh"
```

#### 创建用户 (验证和错误处理)

```bash
# 英文错误消息
curl -X POST -H "Content-Type: application/json" -d '{}' http://localhost:8080/api/v1/users

# 中文错误消息
curl -X POST -H "Content-Type: application/json" -d '{}' "http://localhost:8080/api/v1/users?lang=zh"
```

#### 获取用户问候信息 (复合翻译)

```bash
curl http://localhost:8080/api/v1/users/123/greeting?lang=ja
```

## 优势

### 1. **分层友好**

- 每一层都可以独立使用 i18n 功能
- 不依赖 HTTP 层的特定类型

### 2. **自动传递**

- 通过 context 自动传递，无需手动传递 i18n 实例
- 符合 Go 的最佳实践

### 3. **回退保障**

- 每层都有回退机制，保证代码健壮性
- 即使 context 为空也能正常工作

### 4. **测试友好**

- 在单元测试中可以轻松注入测试用的 i18n 实例
- 不需要模拟 HTTP 请求

## 测试用例示例

```go
func TestUserService(t *testing.T) {
    service := NewUserService()

    // 创建测试用的 i18n 实例
    i18nInstance := i18n.New(
        i18n.WithLanguage(language.English),
        i18n.WithFile("./test_locales"),
    )

    // 创建带有 i18n 的 context
    ctx := i18n.WithContext(context.Background(), i18nInstance)

    // 测试
    user, err := service.GetUser(ctx, 123)
    assert.NoError(t, err)
    assert.NotNil(t, user)
}
```

## 最佳实践

1. **始终使用 context.Context 传递**：在跨层调用时优先使用 context 传递 i18n
2. **提供回退机制**：每层都应该有默认的 i18n 实例作为回退
3. **错误处理**：在适当的层级进行错误翻译，避免重复翻译
4. **测试覆盖**：确保 i18n 功能在各层的测试中得到覆盖
