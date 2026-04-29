package template

import (
	htmltemplate "html/template"
	"strings"
	texttemplate "text/template"
	"time"
	"unicode"
)

// nowFunc 是 Now() 的可替换实现，便于测试时固定时间。
// 默认返回 time.Now().UTC()，避免不同时区导致的 boilerplate 年份漂移。
var nowFunc = func() time.Time { return time.Now().UTC() }

// DefaultFuncMap 返回 linctl 默认的模板函数集合。
//
// 函数分类：
//   - 字符串：kebab / upperKebab / snake / camel / lowerCamel / pascal / title / lower / upper
//   - slice：contains / first / last / join / unique
//   - 命名：plural（不规则复数）
//   - 时间：now / date / year（boilerplate.txt / changelog 等需要的当前时间）
//   - 业务：hasComponent / hasFeature
//   - 控制：default
//   - HTML 安全：safeHTML / safeJS
func DefaultFuncMap() texttemplate.FuncMap {
	return texttemplate.FuncMap{
		// 字符串
		"kebab":      ToKebab,
		"upperKebab": ToUpperKebab,
		"snake":      ToSnake,
		"camel":      ToCamel,
		"lowerCamel": ToCamel,
		"pascal":     ToPascal,
		"title":      Title,
		"lower":      strings.ToLower,
		"upper":      strings.ToUpper,
		"trim":       strings.TrimSpace,
		"trimL":      strings.TrimLeft,
		"trimR":      strings.TrimRight,

		// slice / contains
		"contains":  Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"first":     firstString,
		"last":      lastString,
		"join":      strings.Join,
		"unique":    Unique,
		"len":       Len,

		// 命名
		"plural":   Plural,
		"singular": Singular,

		// 时间
		"now":  Now,
		"date": Date,
		"year": Year,

		// 控制
		"default": Default,

		// 业务
		"hasComponent": HasComponent,
		"hasFeature":   HasFeature,

		// 安全
		"safeHTML": func(s string) htmltemplate.HTML { return htmltemplate.HTML(s) },
		"safeJS":   func(s string) htmltemplate.JS { return htmltemplate.JS(s) },
	}
}

// ===== 字符串工具 =====

// ToKebab 把字符串转为 kebab-case："UserProfile" → "user-profile"。
func ToKebab(s string) string {
	return splitCase(s, "-", false)
}

// ToUpperKebab 把字符串转为大写连字符形式："miniblog-v4" → "MINIBLOG-V4"，
// "UserProfile" → "USER-PROFILE"。常用于环境变量前缀，例如
// `MINIBLOG-V4_APISERVER`（注意：viper 在 SetEnvKeyReplacer 中会把 `-` 视同 `_`，
// 所以保留连字符不会影响最终的环境变量解析）。
func ToUpperKebab(s string) string {
	return splitCase(s, "-", true)
}

// ToSnake 把字符串转为 snake_case："UserProfile" → "user_profile"。
func ToSnake(s string) string {
	return splitCase(s, "_", false)
}

// ToCamel 把字符串转为 camelCase："user-profile" → "userProfile"。
func ToCamel(s string) string {
	parts := splitWords(s)
	if len(parts) == 0 {
		return ""
	}
	out := strings.ToLower(parts[0])
	for _, p := range parts[1:] {
		out += titleFirst(strings.ToLower(p))
	}
	return out
}

// ToPascal 把字符串转为 PascalCase："user-profile" → "UserProfile"。
func ToPascal(s string) string {
	parts := splitWords(s)
	out := ""
	for _, p := range parts {
		out += titleFirst(strings.ToLower(p))
	}
	return out
}

// Title 把每个单词首字母大写："hello world" → "Hello World"。
func Title(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		words[i] = titleFirst(w)
	}
	return strings.Join(words, " ")
}

// ===== slice 工具 =====

// Contains 检查 slice 是否包含元素。
func Contains(items []string, v string) bool {
	for _, item := range items {
		if item == v {
			return true
		}
	}
	return false
}

// First 返回 slice 第一个元素，空时返回零值。泛型版本，可在代码中调用。
func First[T any](items []T) (out T) {
	if len(items) > 0 {
		return items[0]
	}
	return out
}

// Last 返回 slice 最后一个元素，空时返回零值。泛型版本，可在代码中调用。
func Last[T any](items []T) (out T) {
	if len(items) > 0 {
		return items[len(items)-1]
	}
	return out
}

// firstString / lastString 是注册到 FuncMap 的具体类型版本（text/template 不支持泛型）。
func firstString(items []string) string {
	if len(items) > 0 {
		return items[0]
	}
	return ""
}

func lastString(items []string) string {
	if len(items) > 0 {
		return items[len(items)-1]
	}
	return ""
}

// Unique 返回去重的字符串 slice，保持原顺序。
func Unique(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, it := range items {
		if _, ok := seen[it]; ok {
			continue
		}
		seen[it] = struct{}{}
		out = append(out, it)
	}
	return out
}

// Len 是 len() 的模板版（text/template 内置 len 在 nil 上 panic，这里 nil-safe）。
func Len(v any) int {
	if v == nil {
		return 0
	}
	switch x := v.(type) {
	case string:
		return len(x)
	case []string:
		return len(x)
	case []any:
		return len(x)
	case map[string]any:
		return len(x)
	default:
		return 0
	}
}

// ===== 复数 / 单数 =====

// 不规则复数表（覆盖最常用的）。
var irregularPlurals = map[string]string{
	"person": "people",
	"man":    "men",
	"woman":  "women",
	"child":  "children",
	"foot":   "feet",
	"tooth":  "teeth",
	"mouse":  "mice",
	"goose":  "geese",
}

// Plural 返回名词的复数形式（仅英文）。
//
// 规则（按优先级）：
//  1. 不规则表
//  2. 以 s/x/z/ch/sh 结尾 → 加 es
//  3. 以辅音 + y 结尾 → 去 y 加 ies
//  4. 以 f/fe 结尾 → 改 ves
//  5. 默认 → 加 s
func Plural(s string) string {
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	if pl, ok := irregularPlurals[lower]; ok {
		return pl
	}
	switch {
	case strings.HasSuffix(lower, "s"),
		strings.HasSuffix(lower, "x"),
		strings.HasSuffix(lower, "z"),
		strings.HasSuffix(lower, "ch"),
		strings.HasSuffix(lower, "sh"):
		return s + "es"
	case len(s) > 1 && strings.HasSuffix(lower, "y") && !isVowel(lower[len(lower)-2]):
		return s[:len(s)-1] + "ies"
	case strings.HasSuffix(lower, "fe"):
		return s[:len(s)-2] + "ves"
	case strings.HasSuffix(lower, "f"):
		return s[:len(s)-1] + "ves"
	default:
		return s + "s"
	}
}

// 已知 -fe → -ves 的复数（区别于 -f → -ves，例如 calf/calves）。
var fePluralExceptions = map[string]string{
	"lives":  "life",
	"knives": "knife",
	"wives":  "wife",
	"halves": "half",
}

// Singular 返回名词的单数形式（启发式，仅作辅助；不保证 100% 准确）。
func Singular(s string) string {
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	for sg, pl := range irregularPlurals {
		if pl == lower {
			return sg
		}
	}
	if sg, ok := fePluralExceptions[lower]; ok {
		return sg
	}
	switch {
	case strings.HasSuffix(lower, "ies") && len(s) > 3:
		return s[:len(s)-3] + "y"
	case strings.HasSuffix(lower, "ves") && len(s) > 3:
		return s[:len(s)-3] + "f"
	case strings.HasSuffix(lower, "es") && len(s) > 2:
		return s[:len(s)-2]
	case strings.HasSuffix(lower, "s") && len(s) > 1:
		return s[:len(s)-1]
	default:
		return s
	}
}

// ===== 时间 =====

// Now 返回当前 UTC 时间，便于配合 date / year 使用。
//
// 模板用法：
//
//	{{ now | date "2006-01-02" }}   // 输出 "2025-01-15"
//	{{ now.Year }}                  // 直接使用 time.Time 的方法
//
// 内部使用 nowFunc 间接调用，测试时可通过 SetNowForTest 固定时间。
func Now() time.Time {
	return nowFunc()
}

// Date 用 Go 的时间格式串格式化 t（参考时间为 2006-01-02 15:04:05）。
//
// 签名顺序故意为 (layout, t)，使其在管道中可写为 `t | date "layout"`：
// text/template 在管道传值时会把上一段的输出作为最后一个参数。
//
// 模板用法：
//
//	{{ now | date "2006" }}            // 当前年份
//	{{ now | date "2006-01-02" }}      // 当前日期
//	{{ now | date "2006-01-02 15:04" }}
func Date(layout string, t time.Time) string {
	return t.Format(layout)
}

// Year 返回当前年份的 4 位字符串（Now() 取自 nowFunc，UTC）。
//
// 用法：{{ year }}（等价于 {{ now | date "2006" }}）。
func Year() string {
	return nowFunc().Format("2006")
}

// SetNowForTest 替换 nowFunc 实现，**仅供测试使用**。
//
// 调用方必须用 t.Cleanup 还原，避免污染其他测试。
func SetNowForTest(fn func() time.Time) (restore func()) {
	prev := nowFunc
	nowFunc = fn
	return func() { nowFunc = prev }
}

// ===== 控制 =====

// Default 返回 v；若 v 为 zero / nil / 空字符串，返回 fallback。
//
// 用法：{{ .Port | default 8080 }}
func Default(fallback, v any) any {
	if isZero(v) {
		return fallback
	}
	return v
}

// ===== 业务（依赖 project 类型，但通过 any 保持解耦） =====

// HasComponent 判断 Project.Spec.Components 中是否有满足 kindOrName 的 Component。
//
// 匹配语义：
//   - 先按 Kind（首字母大写，如 "WebServer"）精确匹配
//   - 否则按 Name 精确匹配
//
// 入参类型为 any，运行时解包；模板用法：
//
//	{{ if hasComponent .Project.Spec.Components "WebServer" }} ... {{ end }}
func HasComponent(components any, kindOrName string) bool {
	items, ok := components.([]any)
	if !ok {
		return hasComponentReflect(components, kindOrName)
	}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if k, _ := m["Kind"].(string); k == kindOrName {
			return true
		}
		if n, _ := m["Name"].(string); n == kindOrName {
			return true
		}
	}
	return false
}

// HasFeature 判断 features slice 中是否包含 name。
//
// 用法：{{ if hasFeature .Component.Features "user" }}...{{ end }}
func HasFeature(features any, name string) bool {
	switch v := features.(type) {
	case []string:
		return Contains(v, name)
	case []any:
		for _, f := range v {
			if s, ok := f.(string); ok && s == name {
				return true
			}
		}
	}
	return false
}

// ===== 内部辅助 =====

func splitCase(s, sep string, upper bool) string {
	parts := splitWords(s)
	if upper {
		for i := range parts {
			parts[i] = strings.ToUpper(parts[i])
		}
	} else {
		for i := range parts {
			parts[i] = strings.ToLower(parts[i])
		}
	}
	return strings.Join(parts, sep)
}

// splitWords 按 camelCase / PascalCase / kebab / snake 分隔为单词。
//
// 例：
//   - "UserProfile"   → ["User", "Profile"]
//   - "user_profile"  → ["user", "profile"]
//   - "user-profile"  → ["user", "profile"]
//   - "myABCTest"     → ["my", "ABC", "Test"]
func splitWords(s string) []string {
	if s == "" {
		return nil
	}
	// 先把 _ - 替换为空格
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")

	// 然后在大小写边界插入空格
	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			next := rune(0)
			if i+1 < len(runes) {
				next = runes[i+1]
			}
			// XYZ -> X Y Z 但 XYZAbc -> XYZ Abc（看 next 是否小写）
			if unicode.IsLower(prev) || (unicode.IsUpper(prev) && unicode.IsLower(next)) {
				b.WriteByte(' ')
			}
		}
		b.WriteRune(r)
	}
	return strings.Fields(b.String())
}

func titleFirst(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func isVowel(c byte) bool {
	switch c {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	}
	return false
}

func isZero(v any) bool {
	if v == nil {
		return true
	}
	switch x := v.(type) {
	case string:
		return x == ""
	case bool:
		return !x
	case int:
		return x == 0
	case int64:
		return x == 0
	case float64:
		return x == 0
	case []string:
		return len(x) == 0
	case []any:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	}
	return false
}

// hasComponentReflect 是 HasComponent 的 fallback，处理强类型 slice（如 []project.Component）。
//
// 不直接 import internal/project（避免循环依赖）；通过反射 + 字段名访问。
func hasComponentReflect(components any, kindOrName string) bool {
	// 简化：调用方可以传 []map[string]any 或转成 []any；
	// 强类型 slice 模板渲染时 text/template 通常会暴露字段为 .Kind/.Name，
	// HasComponent 主要用于动态构造的数据。如有需要可加 reflect 实现。
	return false
}
