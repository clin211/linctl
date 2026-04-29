# 错误处理使用示例

## 基本使用

### 1. 创建和使用业务错误

```go
package main

import (
    "{{ .Project.Metadata.Module }}/internal/pkg/errno"
    "{{ .Project.Metadata.Module }}/pkg/core"
    "github.com/gin-gonic/gin"
)

// UserHandler 用户处理器
func UserHandler(c *gin.Context) {
    // 场景1：用户不存在
    err := getUserFromDB("12345")
    if err != nil {
        core.WriteBizError(c, errno.ErrUserNotFound)
        return
    }

    // 场景2：用户余额不足（带详细信息）
    balance := getBalance("12345")
    if balance < 100 {
        err := errno.ErrUserInsufficientBalance.WithMetadata(map[string]interface{}{
            "current_balance": balance,
            "required_amount": 100,
            "retry_after":     3600, // 1小时后重试
        })
        core.WriteBizError(c, err)
        return
    }

    // 场景3：操作成功
    data := map[string]string{"user_id": "12345", "status": "active"}
    core.WriteSuccess(c, data, "获取用户信息成功")
}
```

### 2. 自定义业务错误

```go
// 创建自定义错误
ErrCustomBusiness := errorsx.NewBizError(
    errorsx.CodeUserPermissionDenied,
    "User.InsufficientVIP",
    "用户VIP等级不足，无法执行此操作",
)

// 使用链式调用添加额外信息
customErr := ErrCustomBusiness.WithDetails("需要VIP 3级，当前为VIP 1级").
    WithMetadata(map[string]interface{}{
        "current_vip": 1,
        "required_vip": 3,
        "upgrade_url": "https://example.com/vip/upgrade",
    })

// 使用 WithMessage 方法（创建新实例，避免内存共享）
dynamicErr := ErrCustomBusiness.WithMessage("您的VIP等级不足，请升级后重试")

// 此时 ErrCustomBusiness 的消息仍然是原值，dynamicErr 的消息是新值
```

### 3. 在业务逻辑中使用

```go
type UserService struct{}

func (s *UserService) UpdateUser(userID string, req *UpdateUserRequest) error {
    // 检查用户是否存在
    user, err := s.GetUserByID(userID)
    if err != nil {
        if errors.Is(err, errno.ErrUserNotFound) {
            return errno.ErrUserNotFound
        }
        return err // 数据库错误
    }

    // 检查权限
    if !user.IsActive {
        return errorsx.NewBizError(
            errorsx.CodeUserPermissionDenied,
            "User.Inactive",
            "用户账户已被禁用",
        ).WithMetadata(map[string]interface{}{
            "user_id": userID,
            "status":  user.Status,
        })
    }

    // 更新用户信息
    return s.UpdateUserInDB(userID, req)
}
```

### 4. 响应格式示例

#### 成功响应

```json
{
  "code": 0,
  "message": "获取用户信息成功",
  "data": {
    "user_id": "12345",
    "username": "john_doe",
    "email": "john@example.com"
  }
}
```

#### 业务错误响应（简单）

```json
{
  "code": 20101,
  "message": "用户不存在",
  "data": null,
  "reason": "User.NotFound"
}
```

#### 业务错误响应（带元数据）

```json
{
  "code": 20104,
  "message": "用户余额不足，请先充值",
  "data": null,
  "reason": "User.InsufficientBalance"
}
```

元数据（通过 Response Headers 传递）：
```
X-Request-ID: req_123456789
X-Timestamp: 1640995200
X-Response-Time: 150
```

或者在复杂场景下，可以将详细元数据存储在结构化的 Metadata 字段中：

```go
// 对于需要传递复杂元数据的错误
err := errno.ErrUserInsufficientBalance.WithMetadata(map[string]interface{}{
    "current_balance": 0.00,
    "required_amount": 99.99,
    "currency": "CNY",
    "payment_methods": []string{"alipay", "wechat", "credit_card"},
    "promo_code": "SAVE20",
})
```

## Reason 字段使用规范

### 1. 基础格式

```go
// Module.Action 格式
"User.NotFound"           // 用户未找到
"Auth.TokenInvalid"       // Token无效
"Database.ConnectionFailed" // 数据库连接失败
"Cache.KeyNotFound"       // 缓存键未找到
```

### 2. 详细格式

```go
// Module.Action.Detail 格式
"User.NotFound.Inactive"      // 用户未找到（可能已被禁用）
"Auth.TokenExpired.Renewable"  // Token过期但可续期
"Database.Connection.Timeout" // 数据库连接超时
"Payment.Processing.Failed"   // 支付处理失败
```

### 3. 服务相关格式

```go
// Service.Type.Error 格式
"Database.Read.PermissionDenied"   // 数据库读取权限不足
"Cache.Write.SerializationFailed"  // 缓存写入序列化失败
"Payment.Gateway.Timeout"          // 支付网关超时
"Notification.Email.SendFailed"   // 邮件发送失败
```

### 4. 业务流程格式

```go
// BusinessFlow.Step.Error 格式
"Order.Create.InsufficientStock"  // 创建订单库存不足
"Checkout.Payment.Required"        // 结账需要支付
"Withdrawal.DailyLimit.Exceeded"  // 超出每日提现限额
"UserRegistration.Email.Conflict" // 用户注册邮箱冲突
```

## WithMessage 方法内存安全性

### 1. 为什么需要创建新实例？

`WithMessage` 方法创建新的错误实例，避免内存共享问题：

```go
// ❌ 错误方式：如果在同一个实例上修改消息，会影响所有引用
sharedErr := errno.ErrUserNotFound
// 如果 sharedErr.WithMessage() 直接修改原实例，
// 所有使用 sharedErr 的地方都会受到影响

// ✅ 正确方式：创建新实例
originalErr := errno.ErrUserNotFound  // 原实例保持不变
dynamicErr := originalErr.WithMessage("根据上下文定制的错误消息")  // 新实例
```

### 2. 内存复制策略

```go
func (err *BizError) WithMessage(message string) *BizError {
    // 创建完全独立的新实例
    newErr := &BizError{
        Code:     err.Code,      // 值复制
        Level:    err.Level,     // 值复制
        Reason:   err.Reason,    // 值复制
        Message:  message,       // 新值
        Details:  err.Details,    // 值复制
        Metadata: nil,           // 延迟复制
    }

    // 深度复制元数据，避免引用共享
    if err.Metadata != nil {
        newErr.Metadata = make(map[string]interface{})
        for k, v := range err.Metadata {
            newErr.Metadata[k] = v
        }
    }

    return newErr
}
```

### 3. 使用场景

```go
// 场景1：根据具体错误上下文定制消息
baseErr := errno.ErrUserInsufficientBalance

if orderAmount > balance {
    // 使用定制消息
    customizedErr := baseErr.WithMessage(
        fmt.Sprintf("余额不足，当前：¥%.2f，需要：¥%.2f", balance, orderAmount),
    ).WithMetadata(map[string]interface{}{
        "balance": balance,
        "required": orderAmount,
    })
    core.WriteBizError(c, customizedErr)
} else {
    // 使用基础消息
    core.WriteBizError(c, baseErr)
}

// 场景2：在循环中复用基础错误
func processUsers(users []User) error {
    baseErr := errno.ErrUserPermissionDenied

    for _, user := range users {
        if !user.IsActive {
            // 每次都创建新的错误实例，带有用户特定的消息
            err := baseErr.WithMessage(
                fmt.Sprintf("用户 %s 的账户已被禁用", user.Username),
            ).WithMetadata(map[string]interface{}{
                "user_id": user.ID,
                "status": user.Status,
            })
            return err
        }
    }
    return nil
}
```

### 4. 性能考虑

虽然 `WithMessage` 会创建新实例，但在实际应用中：

- 错误通常是异常情况，创建开销可忽略
- 内存复制仅针对 `Metadata`，其他字段为值传递
- 相比内存共享导致的数据污染，性能代价是值得的

## 错误处理最佳实践

### 1. 错误级别选择

```go
// Level 1: 系统错误 - 需要开发介入
CodeInternalServer        // 数据库崩溃、配置错误等

// Level 2: 用户错误 - 可提示用户操作
CodeUserNotFound          // 用户不存在、密码错误等
CodeUserInvalidUsername   // 用户名格式错误

// Level 3: 业务错误 - 正常业务流程
CodeUserInsufficientBalance // 余额不足（可充值）
CodePostAlreadyPublished   // 文章已发布（可编辑）

// Level 4: 上游服务错误
CodePaymentGatewayTimeout   // 支付网关超时

// Level 5: 下游服务错误
CodeEmailServiceUnavailable // 邮件服务不可用
```

### 2. 错误信息设计

```go
// ✅ 好的错误信息
"用户不存在，请检查用户名是否正确"
"密码错误，还剩 3 次尝试机会"
"余额不足，当前余额：¥0.00，需要：¥99.99"

// ❌ 不好的错误信息
"error"          // 太模糊
"操作失败"        // 不够具体
"系统错误"        // 用户无法理解
```

### 3. 元数据使用

```go
// ✅ 有用的元数据
err.WithMetadata(map[string]interface{}{
    "user_id": "12345",           // 用于追踪
    "retry_after": 300,           // 建议重试时间
    "required_role": "admin",     // 需要的角色
    "current_balance": 100,       // 当前余额
    "help_url": "https://...",    // 帮助链接
})

// ❌ 无用的元数据
err.WithMetadata(map[string]interface{}{
    "timestamp": "2023-01-01",    // 应该在Header中
    "server_ip": "10.0.0.1",      // 不暴露给用户
    "debug_info": "stack trace",  // 敏感信息
})
```

### 4. 错误链处理

```go
func (s *UserService) DeleteUser(userID string) error {
    // 检查依赖关系
    orders, err := s.GetOrdersByUser(userID)
    if err != nil {
        return fmt.Errorf("failed to get user orders: %w", err)
    }

    if len(orders) > 0 {
        return errorsx.NewBizError(
            errorsx.CodeUserPermissionDenied,
            "User.HasActiveOrders",
            "用户有未完成的订单，无法删除账户",
        ).WithMetadata(map[string]interface{}{
            "order_count": len(orders),
            "user_id": userID,
        })
    }

    return s.DeleteUserFromDB(userID)
}
```

## 监控和日志集成

### 1. 结构化日志

```go
func (h *UserHandler) DeleteUser(c *gin.Context) {
    userID := c.Param("id")

    err := h.userService.DeleteUser(userID)
    if err != nil {
        // 记录错误日志
        log.Error("delete user failed",
            "user_id", userID,
            "error_code", errorsx.Code(err),
            "reason", errorsx.Reason(err),
            "request_id", c.GetString("request_id"),
        )

        core.WriteBizError(c, err)
        return
    }

    log.Info("user deleted successfully",
        "user_id", userID,
        "request_id", c.GetString("request_id"),
    )

    core.WriteSuccess(c, nil, "用户删除成功")
}
```

### 2. 指标收集

```go
func ErrorMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()

        if len(c.Errors) > 0 {
            err := c.Errors.Last()
            errorCode := errorsx.Code(err)

            // 记录错误指标
            errorCounter.WithLabelValues(
                strconv.Itoa(errorCode),
                errorsx.Reason(err),
                c.FullPath(),
            ).Inc()
        }
    }
}
```

这样设计既保持了灵活性，又确保了错误处理的一致性和可维护性。reason 字段的格式不固化，可以根据实际业务需要进行扩展和调整。
