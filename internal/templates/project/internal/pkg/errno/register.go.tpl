package errno

// RegisterErrors registers a set of BizErrors with the application.
// Implementations can extend this to publish to a registry or metrics system.
func RegisterErrors(errs ...*BizError) {
	_ = errs // placeholder: extend as needed
}

// RegisterAll is called once at application startup to register all resource errors.
//
// `linctl add <Resource> --with errno` appends new RegisterErrors(<Resource>Errors()...)
// statements to this function body via AST.
func RegisterAll() {
}
