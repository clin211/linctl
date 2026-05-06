package v1

// {{.Resource | Pascal}}Info 表示一个 {{.Resource | Pascal}} 实体。
type {{.Resource | Pascal}}Info struct {
	{{.Resource | Pascal}}Id string `json:"{{.Resource | LowerCamel}}Id"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

// Create{{.Resource | Pascal}}Request 是 Create{{.Resource | Pascal}} 的请求体。
type Create{{.Resource | Pascal}}Request struct{}

// Create{{.Resource | Pascal}}Response 是 Create{{.Resource | Pascal}} 的响应体。
type Create{{.Resource | Pascal}}Response struct {
	{{.Resource | Pascal}}Id string `json:"{{.Resource | LowerCamel}}Id"`
}

// Update{{.Resource | Pascal}}Request 是 Update{{.Resource | Pascal}} 的请求体。
type Update{{.Resource | Pascal}}Request struct {
	{{.Resource | Pascal}}Id string `json:"{{.Resource | LowerCamel}}Id"`
}

// Update{{.Resource | Pascal}}Response 是 Update{{.Resource | Pascal}} 的响应体。
type Update{{.Resource | Pascal}}Response struct{}

// Delete{{.Resource | Pascal}}Request 是 Delete{{.Resource | Pascal}} 的请求体。
type Delete{{.Resource | Pascal}}Request struct {
	{{.Resource | Pascal}}Id string `json:"{{.Resource | LowerCamel}}Id"`
}

// Delete{{.Resource | Pascal}}Response 是 Delete{{.Resource | Pascal}} 的响应体。
type Delete{{.Resource | Pascal}}Response struct{}

// Get{{.Resource | Pascal}}Request 是 Get{{.Resource | Pascal}} 的请求体。
type Get{{.Resource | Pascal}}Request struct {
	{{.Resource | Pascal}}Id string `json:"{{.Resource | LowerCamel}}Id"`
}

// Get{{.Resource | Pascal}}Response 是 Get{{.Resource | Pascal}} 的响应体。
type Get{{.Resource | Pascal}}Response struct {
	{{.Resource | Pascal}} *{{.Resource | Pascal}}Info `json:"{{.Resource | LowerCamel}}"`
}

// List{{.Resource | Pascal}}Request 是 List{{.Resource | Pascal}} 的请求体。
type List{{.Resource | Pascal}}Request struct {
	Offset int64 `json:"offset" form:"offset"`
	Limit  int64 `json:"limit" form:"limit"`
}

// List{{.Resource | Pascal}}Response 是 List{{.Resource | Pascal}} 的响应体。
type List{{.Resource | Pascal}}Response struct {
	Total  int64               `json:"total"`
	{{.Resource | Pascal | Plural}} []*{{.Resource | Pascal}}Info `json:"{{.Resource | LowerCamel | Plural}}"`
}
