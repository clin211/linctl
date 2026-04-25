# ADR-003: Proto 文件 AST 操作用 `bufbuild/protocompile` 替代字符串行扫描

## 元数据

| 字段 | 值 |
| --- | --- |
| **状态** | ✅ Accepted (2026-04-25) |
| **日期** | 2026-04-25 |
| **作者** | @clin211 |
| **审阅人** | TBD |
| **相关 ADR** | [ADR-001](./001-use-dst-not-goast.md)（同样的"AST 思路"） |
| **影响范围** | `internal/ast/proto_inject.go`、`07-ast-injection.md` |
| **相关 issue/PR** | - |

---

## 1. 背景

osbuilder 修改 `.proto` 文件用**字符串行扫描**：

```go
for i, line := range lines {
    if strings.Contains(line, "service "+grpcServiceName+" {") {
        // 在这一行后追加 5 个 rpc method
    }
}
```

问题：

| 场景 | 出错表现 |
| --- | --- |
| 注释里写了 `// service Foo {` | 误判，在错误位置插入 |
| 多个 service 块在同一文件 | 总是插入到第一个 |
| 文件含 gRPC streaming 修饰符 (`stream Request`) | 不识别 |
| 含 `google.api.http` annotation | 行匹配失败 |
| 已有同名方法 | 重复插入 |
| 文件以 CRLF 结尾 | line index 错乱 |

为保证 `linctl add api` 等命令对**任意合法 .proto 文件**都能正确修改，必须使用真正的 Proto AST 解析器。

## 2. 决策

我们决定用 [`github.com/bufbuild/protocompile`](https://github.com/bufbuild/protocompile) 解析 .proto，再做修改。

关键变更：

1. **解析阶段**：用 `protocompile/parser` 把文件 parse 成 `*descriptorpb.FileDescriptorProto`。
2. **检查阶段**：在 descriptor 上查找目标 service / 已有 methods（精确匹配，无误判）。
3. **修改阶段**：
   - **Phase 1**（保留原始格式）：基于 AST 定位 service 块的字节偏移 → 直接在文本上插入新 method 字符串。
   - **Phase 4**（远期）：完全用 descriptor 重写文件 + `buf format` 美化。
4. **格式化**：可选调用 `buf format` 美化输出（如果用户安装了 buf）。

## 3. 备选方案

| # | 方案 | 优势 | 劣势 | 否决原因 |
| --- | --- | --- | --- | --- |
| 1 | **`bufbuild/protocompile`**（chosen） | buf.build 官方维护；性能好；API 现代 | 文档少；descriptor → text 需自实现 | - |
| 2 | `jhump/protoreflect` | 流行；descriptor → text 内置 | API 偏老；维护节奏慢 | 长期不如 buf.build 系 |
| 3 | `protoc --print_free_field_numbers` 调子进程 | 用户已有 protoc | 串接子进程慢；返回值难解析 | 工程性差 |
| 4 | 字符串扫描（osbuilder 现状） | 零依赖 | 上述所有问题 | 已被验证不可行 |
| 5 | 自研 Proto 词法分析器 | 完全可控 | Proto 语法复杂（service / oneof / map / 嵌套 message / option），自研 ROI 极低 | 性价比差 |

## 4. 后果

### 4.1 正面

- **正确性**：所有合法 .proto 文件都能正确修改，没有 false positive / false negative。
- **重复检测**：基于 descriptor 的精确字段比对，绝不重复插入。
- **错误信息**：parse 失败时给出"文件 + 行号 + 期望"的标准错误，便于用户修复。
- **未来兼容**：buf.build 是 Protobuf 生态主流，长期维护有保障。

### 4.2 负面 / Trade-offs

- **直接依赖增加**：`protocompile` + 间接的 `google.golang.org/protobuf`，约 2-3 MB（已有 grpc 用户基本无负担）。
- **descriptor → text 不内置**：Phase 1 需要自己实现"在原始文本上做局部插入"，逻辑相对复杂。
- **学习曲线**：开发者要理解 Protobuf descriptor 的层次结构。

### 4.3 中性

- 二进制大小增加约 **2-3 MB**（含 google.golang.org/protobuf）。
- 编译时间几乎无影响。

## 5. 渐进实施策略

为了控制 Phase 1 的复杂度，分两步走：

### Phase 1：保留原始格式 + 文本插入

```go
// 1) 用 protocompile 解析得到 service 块的字节偏移
// 2) 在偏移位置插入新 method 文本
// 3) 用 buf format（如果可用）美化
```

优点：保留所有原始注释 / 缩进 / 空行；改动最小。  
缺点：依赖文本操作，对极端格式仍有 edge case。

### Phase 4：完全重写

```go
// 1) 用 protocompile 解析整个文件
// 2) 在 descriptor 上做修改
// 3) 用模板 + descriptor 数据重写整个文件
// 4) buf format 美化
```

优点：完全脱离原始文本，最稳健。  
缺点：丢失用户的注释（除非引入 source code info 的特殊处理）。

## 6. 风险缓解

- **buf 二进制可选**：runtime 检测 `which buf`，未安装时跳过 format（功能仍可用，只是输出不那么美观）。
- **抽象层**：与 [ADR-001](./001-use-dst-not-goast.md) 同思路，所有 Proto 操作封装在 `internal/ast/proto_inject.go` 的 `AddProtoRPCMutator` 等类型下。
- **回归测试**：每个 mutator 都有 golden file 测试，覆盖：单 service / 多 service / streaming / annotation / 嵌套 message / 重复检测。

## 7. 参考资料

- [bufbuild/protocompile GitHub](https://github.com/bufbuild/protocompile)
- [Protobuf Descriptor 文档](https://protobuf.dev/programming-guides/dos-donts/)
- [设计文档：07-ast-injection.md §7.6](../07-ast-injection.md)

---

_Last reviewed: 2026-04-25_
