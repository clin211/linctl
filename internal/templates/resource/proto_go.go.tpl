// Code generated placeholder by lin. DO NOT EDIT manually.
// This file provides placeholder types until `make protoc` generates the real types.
// After running `make protoc`, delete this file (the generated .pb.go files replace it).
package v1

// {{.Resource | Pascal}}Info represents a {{.Resource | Pascal}} entity.
type {{.Resource | Pascal}}Info struct {
	{{.Resource | Pascal}}Id string `json:"{{.Resource | LowerCamel}}Id"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

// Create{{.Resource | Pascal}}Request is the request for Create{{.Resource | Pascal}}.
type Create{{.Resource | Pascal}}Request struct{}

// Create{{.Resource | Pascal}}Response is the response for Create{{.Resource | Pascal}}.
type Create{{.Resource | Pascal}}Response struct {
	{{.Resource | Pascal}}Id string `json:"{{.Resource | LowerCamel}}Id"`
}

// Update{{.Resource | Pascal}}Request is the request for Update{{.Resource | Pascal}}.
type Update{{.Resource | Pascal}}Request struct {
	{{.Resource | Pascal}}Id string `json:"{{.Resource | LowerCamel}}Id"`
}

// Update{{.Resource | Pascal}}Response is the response for Update{{.Resource | Pascal}}.
type Update{{.Resource | Pascal}}Response struct{}

// Delete{{.Resource | Pascal}}Request is the request for Delete{{.Resource | Pascal}}.
type Delete{{.Resource | Pascal}}Request struct {
	{{.Resource | Pascal}}Id string `json:"{{.Resource | LowerCamel}}Id"`
}

// Delete{{.Resource | Pascal}}Response is the response for Delete{{.Resource | Pascal}}.
type Delete{{.Resource | Pascal}}Response struct{}

// Get{{.Resource | Pascal}}Request is the request for Get{{.Resource | Pascal}}.
type Get{{.Resource | Pascal}}Request struct {
	{{.Resource | Pascal}}Id string `json:"{{.Resource | LowerCamel}}Id"`
}

// Get{{.Resource | Pascal}}Response is the response for Get{{.Resource | Pascal}}.
type Get{{.Resource | Pascal}}Response struct {
	{{.Resource | Pascal}} *{{.Resource | Pascal}}Info `json:"{{.Resource | LowerCamel}}"`
}

// List{{.Resource | Pascal}}Request is the request for List{{.Resource | Pascal}}.
type List{{.Resource | Pascal}}Request struct {
	Offset int64 `json:"offset" form:"offset"`
	Limit  int64 `json:"limit" form:"limit"`
}

// List{{.Resource | Pascal}}Response is the response for List{{.Resource | Pascal}}.
type List{{.Resource | Pascal}}Response struct {
	Total  int64               `json:"total"`
	{{.Resource | Pascal | Plural}} []*{{.Resource | Pascal}}Info `json:"{{.Resource | LowerCamel | Plural}}"`
}
