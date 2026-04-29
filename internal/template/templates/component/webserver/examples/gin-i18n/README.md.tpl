# Gin i18n 示例

这是一个使用 Gin 框架和 {{ .Project.Metadata.Name }} 的 i18n 包构建的国际化示例项目。

## 功能特性

- 🌍 多语言支持 (英语、中文、日语)
- 🔧 多种语言切换方式 (URL参数、Cookie、Accept-Language头)
- 🎯 Gin 中间件集成
- 📝 JSON 格式的翻译文件
- 🚀 简单易用的 API

## 项目结构

```
gin-i18n/
├── main.go              # 主程序入口
├── go.mod              # Go 模块文件
├── locales/            # 翻译文件目录
│   ├── en.json         # 英语翻译
│   ├── zh.json         # 中文翻译
│   └── ja.json         # 日语翻译
├── middleware/         # 中间件
│   └── i18n.go         # i18n 中间件实现
├── handlers/           # 处理器
│   └── example.go      # 示例处理器
└── README.md           # 说明文档
```

## 快速开始

### 1. 安装依赖

```bash
cd examples/gin-i18n
go mod tidy
```

### 2. 运行程序

```bash
go run main.go
```

服务器将在 `http://localhost:8080` 启动。

### 3. 测试 API

#### 基础问候

```bash
# 默认英语
curl http://localhost:8080/api/v1/hello

# 切换到中文
curl "http://localhost:8080/api/v1/hello?lang=zh"

# 切换到日语
curl "http://localhost:8080/api/v1/hello?lang=ja"
```

#### 用户相关消息

```bash
curl http://localhost:8080/api/v1/user
```

#### 错误消息

```bash
# 不同类型的错误消息
curl "http://localhost:8080/api/v1/error?type=login_required"
curl "http://localhost:8080/api/v1/error?type=invalid_credentials"
curl "http://localhost:8080/api/v1/error?type=user_not_found"
```

#### 数量相关的翻译

```bash
# 不同数量的项目
curl "http://localhost:8080/api/v1/items?count=0"
curl "http://localhost:8080/api/v1/items?count=1"
curl "http://localhost:8080/api/v1/items?count=5"
```

#### 语言管理

```bash
# 查看当前语言信息
curl http://localhost:8080/api/v1/language

# 设置语言偏好 (通过 Cookie)
curl -X POST http://localhost:8080/set-language/zh

# 使用 Accept-Language 头
curl -H "Accept-Language: zh-CN" http://localhost:8080/api/v1/hello
```

#### 健康检查

```bash
curl http://localhost:8080/healthz
```

## API 端点

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/hello` | 基础问候消息 |
| GET | `/api/v1/user` | 用户相关消息 |
| GET | `/api/v1/error` | 错误消息演示 |
| GET | `/api/v1/items` | 数量相关翻译演示 |
| GET | `/api/v1/language` | 当前语言信息 |
| POST | `/set-language/:lang` | 设置语言偏好 |
| GET | `/healthz` | 健康检查 |

## 语言切换方式

### 1. URL 参数 (优先级最高)

```bash
curl "http://localhost:8080/api/v1/hello?lang=zh"
```

支持的语言代码：
- `en` - 英语
- `zh` - 中文
- `ja` - 日语

### 2. Cookie

```bash
# 设置语言偏好
curl -X POST http://localhost:8080/set-language/zh

# 后续请求会自动使用 Cookie 中的语言
curl http://localhost:8080/api/v1/hello
```

### 3. Accept-Language 头

```bash
curl -H "Accept-Language: zh-CN,zh;q=0.9,en;q=0.8" http://localhost:8080/api/v1/hello
```

### 4. 默认语言

如果以上方式都没有指定语言，将使用 i18n 实例的默认语言 (英语)。

## 翻译文件格式

翻译文件使用 JSON 格式，位于 `locales/` 目录下：

```json
{
  "greeting": {
    "hello": "Hello, World!",
    "welcome": "Welcome to our application"
  },
  "user": {
    "login_required": "Login required to access this resource",
    "invalid_credentials": "Invalid username or password"
  },
  "common": {
    "success": "Operation completed successfully",
    "error": "An error occurred"
  }
}
```

## 代码示例

### 中间件使用

```go
// 创建 i18n 实例
i18nInstance := miniblogI18n.New(
    miniblogI18n.WithLanguage(language.English),
    miniblogI18n.WithFormat("json"),
    miniblogI18n.WithFile("./locales"),
)

// 添加中间件
router.Use(middleware.I18nMiddleware(i18nInstance))
```

### 在处理器中使用

```go
func (h *Handler) HelloHandler(c *gin.Context) {
    // 获取 i18n 实例
    i18nInstance := middleware.GetI18n(c)

    // 翻译文本
    message := i18nInstance.T("greeting.hello")

    c.JSON(200, gin.H{
        "message": message,
        "lang": i18nInstance.Language().String(),
    })
}
```

### 简化翻译

```go
// 使用助手函数
func (h *Handler) ExampleHandler(c *gin.Context) {
    message := middleware.T(c, "common.success")
    c.JSON(200, gin.H{"message": message})
}
```

## 扩展功能

### 添加新语言

1. 在 `locales/` 目录下添加新的翻译文件，如 `fr.json`
2. 在客户端请求时使用对应的语言代码：`?lang=fr`

### 添加新翻译

1. 在所有语言文件中添加相同的键值对
2. 在代码中使用 `i18nInstance.T("your.new.key")` 进行翻译

### 模板数据

对于需要动态数据的翻译，可以使用模板语法：

```json
{
  "welcome_user": "Hello {{ "{{" }}.Name{{ "}}" }}, welcome back!"
}
```

## 注意事项

- 翻译文件热加载需要重启服务器
- 建议在生产环境中将翻译文件嵌入到二进制中
- 语言检测优先级：URL参数 > Cookie > Accept-Language > 默认语言

## 依赖

- [gin-gonic/gin](https://github.com/gin-gonic/gin) - HTTP Web 框架
- [golang.org/x/text](https://pkg.go.dev/golang.org/x/text) - 国际化文本处理
- [{{ .Project.Metadata.Module }}/pkg/i18n](../../pkg/i18n) - i18n 核心包

> 注意：`pkg/i18n` 是项目内的 i18n 抽象包。如果你的项目中尚未实现这个包，可以参考 [go-i18n/v2](https://github.com/nicksnyder/go-i18n) 自行实现，或将本示例的 `pkg/i18n` 引用替换为你自己的 i18n 库。
