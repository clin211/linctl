# ADR-000: <一句话决策标题>

> **复制此模板**：`cp adr/000-template.md adr/<编号>-<short-name>.md`，然后修改下面所有 ` <尖括号> ` 占位符。  
> 模板本身**不计入正式 ADR**，仅供参考。

## 元数据

| 字段 | 值 |
| --- | --- |
| **状态（Status）** | `Proposed` / `Accepted` / `Rejected` / `Deprecated` / `Superseded by ADR-XXX` |
| **日期（Date）** | `YYYY-MM-DD`（决策达成日） |
| **作者（Authors）** | `@github-handle` |
| **审阅人（Reviewers）** | `TBD`（accept 前填写 `@reviewer1`, `@reviewer2`） |
| **相关 ADR** | `ADR-XXX` / 无 |
| **影响范围** | `internal/xxx/` 等具体路径 |
| **相关 issue/PR** | `#NN` |

---

## 1. 背景（Context）

> 一段话描述**当前的问题或需求**，要让外人也能理解为什么需要做这个决策。

举例：
- 我们目前用 X 方式做 Y。
- 它的问题是 A、B、C。
- 我们希望达到 D 的效果。

---

## 2. 决策（Decision）

> 用一句话表述**我们决定怎么做**，然后展开关键细节（≤ 200 字）。

```text
我们决定用 <方案 X> 替代 <方案 Y>。
关键变更：
- ...
- ...
```

---

## 3. 备选方案（Alternatives Considered）

| # | 方案 | 优势 | 劣势 | 否决原因 |
| --- | --- | --- | --- | --- |
| 1 | （我们选的） | ... | ... | （这是 chosen，不需要否决原因） |
| 2 | 备选 A | ... | ... | ... |
| 3 | 备选 B | ... | ... | ... |

> 至少列出 2 个备选方案。如果只有一个方案可选，说明这个决策"没什么决策可做"，那它不属于 ADR。

---

## 4. 后果（Consequences）

### 4.1 正面（Positive）
- ...
- ...

### 4.2 负面（Negative / Trade-offs）
- ...
- ...

### 4.3 中性（Neutral，仅记录变化）
- 增加了一个外部依赖：`xxx`
- 二进制大小预估增加 ~XXX KB

---

## 5. 实施计划（Implementation Notes，可选）

如果决策需要实施步骤，可在此简述：

1. ...
2. ...
3. 验证标准：...

---

## 6. 参考资料（References）

- [外部链接 1](https://example.com)
- [相关讨论 issue](#)
- [设计文档：XX](../XX-doc.md)

---

> 本模板基于 [Michael Nygard 的 ADR 模板](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions) + linctl 项目实践改造。

---

_Last reviewed: 2026-04-25_
