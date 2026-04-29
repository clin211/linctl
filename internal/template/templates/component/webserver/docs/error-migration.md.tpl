# 错误处理重构迁移指南

## 概述

本次重构将原有的基于HTTP状态码的错误处理体系升级为结构化的业务错误码体系，主要解决了以下问题：

1. **前端兼容性**：业务错误不再返回4xx状态码，避免浏览器控制台报错
2. **错误分类**：采用五级错误码体系（级别+模块+具体错误）
3. **响应优化**：将request_id、timestamp等元数据移至HTTP Header
4. **向后兼容**：保持旧版本API的兼容性

## 新的错误码体系

### 错误码格式

错误码采用5位数字格式：`LLMMMNNN`

- **LL**: 错误级别（1-5）
- **MMM**: 业务模块（001-999）
- **NNN**: 具体错误（001-999）

### 错误级别定义

```go
LevelSystem     = 1 // 系统级错误，需要开发介入
LevelUser       = 2 // 用户操作错误，可提示用户
LevelBusiness   = 3 // 业务逻辑错误，正常的业务流程
LevelUpstream   = 4 // 上游服务错误
LevelDownstream = 5 // 下游服务错误
```

### 业务模块定义

```go
ModuleCommon   = 1 // 通用模块
ModuleUser     = 2 // 用户模块
ModulePost     = 3 // 博客文章模块
ModuleComment  = 4 // 评论模块
ModuleAuth     = 5 // 认证模块
ModuleDatabase = 6 // 数据库模块
ModuleCache    = 7 // 缓存模块
```

### HTTP状态码映射

- **业务错误**（Level 2,3）：统一返回 HTTP 200
- **系统错误**（Level 1）：返回 HTTP 500
- **上下游错误**（Level 4,5）：返回 HTTP 502

## API响应格式

### 标准响应格式

成功响应：
```json
{
  "code": 0,
  "message": "success",
  "data": { /* 业务数据 */ }
}
```

业务错误响应：
```json
{
  "code": 20101,
  "message": "用户不存在",
  "data": null,
  "reason": "User.NotFound"
}
```

### HTTP Headers

所有响应都会包含以下标准头：

```http
X-Request-ID: req_123456789
X-Timestamp: 1640995200
X-Response-Time: 123
X-Server-ID: server-01
X-Trace-ID: trace-123
```

## 使用方式

### 1. 导入包

```go
import (
    "{{ .Project.Metadata.Module }}/pkg/errorsx"
    "{{ .Project.Metadata.Module }}/internal/pkg/errno"
)
```

### 2. 创建业务错误

```go
// 方式1：使用预定义错误
err := errno.ErrUserNotFound

// 方式2：使用错误码创建
err := errorsx.NewBizError(
    errorsx.CodeUserNotFound,
    "User.NotFound",
    "用户不存在",
)

// 方式3：链式调用
err := errorsx.NewBizError(errorsx.CodeUserNotFound, "User.NotFound", "用户不存在").
    WithDetails("用户ID: 12345 不存在").
    WithMetadata(map[string]interface{}{
        "user_id": 12345,
    }).
    WithRetryAfter(300)
```

### 3. 返回响应

```go
// 在 handler 中使用
func GetUser(c *gin.Context) {
    // ... 业务逻辑
    if userNotFound {
        core.WriteBizError(c, errno.ErrUserNotFound)
        return
    }

    core.WriteSuccess(c, user)
}

// 或者使用传统方式
func GetUser(c *gin.Context) {
    data, err := userService.GetUser(id)
    core.WriteResponse(c, data, err)
}
```

### 4. 中间件配置

```go
// 添加响应中间件
router.Use(core.ResponseMiddleware())

// CORS 配置
config := cors.DefaultConfig()
config.ExposeHeaders = []string{
    "X-Request-ID",
    "X-Timestamp",
    "X-Response-Time",
    "X-Server-ID",
    "X-Trace-ID",
}
router.Use(cors.New(config))
```

## 错误码对照表

### 用户模块错误

| 错误码 | 描述 | 原HTTP码 | 新HTTP码 |
|--------|------|----------|----------|
| 20101 | 用户不存在 | 404 | 200 |
| 20102 | 用户已存在 | 400 | 200 |
| 20103 | 用户名或密码错误 | 401 | 200 |
| 20104 | 用户余额不足 | 400 | 200 |
| 20105 | 用户名无效 | 400 | 200 |
| 20106 | 密码无效 | 400 | 200 |
| 20107 | 用户权限不足 | 403 | 200 |

### 认证模块错误

| 错误码 | 描述 | 原HTTP码 | 新HTTP码 |
|--------|------|----------|----------|
| 50101 | 未认证 | 401 | 200 |
| 50102 | Token无效 | 401 | 200 |
| 50103 | Token过期 | 401 | 200 |
| 50104 | Token签名失败 | 500 | 500 |

### 关于 reason 字段

reason 字段表示错误的原因，用于日志和监控，格式相对灵活：

- **基础格式**：`Module.Action` (如 `User.NotFound`, `Auth.TokenInvalid`)
- **详细格式**：`Module.Action.Detail` (如 `User.NotFound.Inactive`)
- **服务相关**：`Service.Type.Error` (如 `Database.Connection.Timeout`)

reason 字段的目的是：
- 提供开发者和运维人员可理解的错误标识
- 便于日志分析和错误统计
- 与具体业务场景结合，可灵活扩展

## 前端适配

### 请求拦截器

```typescript
// 添加请求ID
axios.interceptors.request.use((config) => {
    config.headers['X-Request-ID'] = generateUUID();
    return config;
});

// 响应拦截器
axios.interceptors.response.use(
    (response) => {
        const { data, headers } = response;

        // 获取元数据
        const meta = {
            request_id: headers['x-request-id'],
            timestamp: headers['x-timestamp'],
            response_time: headers['x-response-time'],
            server_id: headers['x-server-id'],
        };

        if (data.code === 0) {
            return { ...data, meta };
        }

        // 业务错误处理
        handleBusinessError(data, meta);
        return Promise.reject({ ...data, meta });
    },
    (error) => {
        // HTTP错误处理
        const meta = {
            request_id: error.response?.headers['x-request-id'],
        };
        return Promise.reject({ ...error, meta });
    }
);
```

## 迁移步骤

### 1. 现有代码兼容

- 旧的 `errorsx.ErrorX` 仍然可用，但标记为 Deprecated
- `errorsx.FromError` 会自动将旧错误转换为新的 `BizError`
- 建议逐步迁移到新的错误类型

### 2. 逐步迁移

1. **第一步**：在新功能中使用新的错误码体系
2. **第二步**：逐步将现有错误替换为新版本
3. **第三步**：更新前端处理逻辑
4. **第四步**：移除旧版本的兼容代码

### 3. 测试建议

```go
func TestErrorMigration(t *testing.T) {
    // 测试旧版本错误转换
    oldErr := &errorsx.ErrorXCompat{
        Code:    404,
        Reason:  "User.NotFound",
        Message: "User not found.",
    }

    bizErr := errorsx.FromError(oldErr)
    assert.Equal(t, errorsx.CodeUserNotFound, bizErr.Code)
    assert.Equal(t, "User.NotFound", bizErr.Reason)
}
```

## 注意事项

1. **性能优化**：将元数据移至Header减少了响应体大小
2. **监控集成**：request_id 贯穿整个请求链路，便于问题追踪
3. **错误统计**：建议将错误码与监控系统（如Prometheus）集成
4. **国际化**：error.message 支持中文，error.reason 用于日志和监控
5. **向后兼容**：确保现有API调用方不受影响
