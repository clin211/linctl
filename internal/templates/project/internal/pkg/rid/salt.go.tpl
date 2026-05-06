package rid

// Salt 返回 rid 生成所使用的固定 salt。
//
// 占位实现：返回常量 1。建议在生产环境中替换为更安全的方案，
// 如从环境变量读取或基于配置注入。
func Salt() uint64 {
	return 1
}
