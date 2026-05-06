package errno

// RegisterErrors 将一组 BizError 注册到应用中。
// 实现方可以扩展此函数，将错误发布到注册中心或指标系统。
func RegisterErrors(errs ...*BizError) {
	_ = errs // 占位实现：可按需扩展
}

// RegisterAll 在应用启动时调用一次，注册所有资源错误。
//
// `linctl add <Resource> --with errno` 通过 AST 向该函数体追加
// `RegisterErrors(<Resource>Errors()...)` 等语句。
func RegisterAll() {
}
