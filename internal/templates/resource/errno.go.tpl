package errno

import "net/http"

// {{.Resource | Pascal}} 业务错误集合。
var (
	Err{{.Resource | Pascal}}NotFound = New(http.StatusNotFound, 0, "{{.Resource | Pascal}}.NotFound", "{{.Resource | Pascal}} not found.")
	Err{{.Resource | Pascal}}Exists   = New(http.StatusConflict, 0, "{{.Resource | Pascal}}.AlreadyExists", "{{.Resource | Pascal}} already exists.")
	Err{{.Resource | Pascal}}Invalid  = New(http.StatusBadRequest, 0, "{{.Resource | Pascal}}.Invalid", "{{.Resource | Pascal}} is invalid.")
)

// {{.Resource | Pascal}}Errors 返回所有与 {{.Resource | Pascal}} 相关的错误，便于统一注册。
func {{.Resource | Pascal}}Errors() []*BizError {
	return []*BizError{Err{{.Resource | Pascal}}NotFound, Err{{.Resource | Pascal}}Exists, Err{{.Resource | Pascal}}Invalid}
}
