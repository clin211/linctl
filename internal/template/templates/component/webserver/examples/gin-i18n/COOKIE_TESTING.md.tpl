# Cookie 测试说明

## 问题分析

当你使用 `curl` 命令测试 i18n 示例时，可能会发现 Cookie 似乎不工作：

```bash
# ❌ 这样测试不会工作
curl -X POST http://localhost:8080/set-language/zh
curl http://localhost:8080/api/v1/hello  # 仍然显示英语
```

**原因**：curl 默认不会保存和发送 Cookie。每个 curl 请求都是独立的。

## 正确的测试方法

### 方法 1：使用 curl 的 Cookie Jar

```bash
# 设置语言并保存 Cookie
curl -c cookies.txt -X POST http://localhost:8080/set-language/zh

# 使用保存的 Cookie 进行请求
curl -b cookies.txt http://localhost:8080/api/v1/hello
```

### 方法 2：在单次请求中设置和读取

```bash
# 测试 URL 参数语言切换
curl "http://localhost:8080/api/v1/hello?lang=zh"    # 中文
curl "http://localhost:8080/api/v1/hello?lang=ja"    # 日语
curl "http://localhost:8080/api/v1/hello?lang=en"    # 英语
```

### 方法 3：使用浏览器

在浏览器中访问：
1. `http://localhost:8080/set-language/zh` - 设置语言
2. `http://localhost:8080/api/v1/hello` - 查看效果

浏览器会自动处理 Cookie。

### 方法 4：使用 Accept-Language 头

```bash
curl -H "Accept-Language: zh-CN" http://localhost:8080/api/v1/hello
curl -H "Accept-Language: ja" http://localhost:8080/api/v1/hello
curl -H "Accept-Language: en-US" http://localhost:8080/api/v1/hello
```

## 完整测试示例

```bash
# 1. 启动服务器
cd examples/gin-i18n
go run main.go

# 2. 测试 URL 参数切换
curl "http://localhost:8080/api/v1/hello?lang=zh"
# 输出：{"data":{"current_lang":"zh","hello":"你好，世界！",...}}

# 3. 测试 Cookie 切换
curl -c cookies.txt -X POST http://localhost:8080/set-language/ja
curl -b cookies.txt http://localhost:8080/api/v1/hello
# 输出：{"data":{"current_lang":"ja","hello":"こんにちは、世界！",...}}

# 4. 测试 Accept-Language 头
curl -H "Accept-Language: zh-CN" http://localhost:8080/api/v1/hello
# 输出：{"data":{"current_lang":"zh","hello":"你好，世界！",...}}
```

## 语言切换优先级

i18n 中间件按以下优先级检测语言：

1. **URL 查询参数** `?lang=zh` (最高优先级)
2. **Cookie** `lang=zh`
3. **Accept-Language 头** `Accept-Language: zh-CN`
4. **默认语言** (英语)

## 调试技巧

如果遇到问题，可以使用调试版本查看详细信息：

```bash
# 启动调试服务器
go run debug.go

# 测试时会看到详细的请求信息，包括 Cookie 和 Headers
curl -c cookies.txt -X POST http://localhost:8081/set-lang/zh
curl -b cookies.txt http://localhost:8081/test
```

## 常见问题

### Q: 为什么 Cookie 不工作？
A: curl 默认不保存 Cookie。使用 `-c` 保存，`-b` 读取。

### Q: 如何设置 Cookie 永久有效？
A: 修改 `SetLanguageHandler` 中的过期时间：
```go
c.SetCookie("lang", lang, 3600*24*365, "/", "", false, false) // 1年
```

### Q: 可以在 JavaScript 中使用吗？
A: 可以！前端可以通过 fetch 或 axios 设置和发送 Cookie。
