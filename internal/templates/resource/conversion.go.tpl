package conversion

import (
	"{{.Module}}/internal/{{.AppName}}/model"
	v1 "{{.Module}}/pkg/api/{{.AppName}}/v1"
)

// {{.Resource | Pascal}}MToProto converts a {{.Resource | Pascal}}M model to a proto {{.Resource | Pascal}}Info.
func {{.Resource | Pascal}}MToProto(m *model.{{.Resource | Pascal}}M) *v1.{{.Resource | Pascal}}Info {
	return &v1.{{.Resource | Pascal}}Info{
		{{.Resource | Pascal}}Id: m.{{.Resource | Pascal}}ID,
		CreatedAt: m.CreatedAt.Unix(),
		UpdatedAt: m.UpdatedAt.Unix(),
	}
}

// Proto{{.Resource | Pascal}}ToM converts a proto {{.Resource | Pascal}}Info to a {{.Resource | Pascal}}M model.
func Proto{{.Resource | Pascal}}ToM(p *v1.{{.Resource | Pascal}}Info) *model.{{.Resource | Pascal}}M {
	return &model.{{.Resource | Pascal}}M{
		{{.Resource | Pascal}}ID: p.{{.Resource | Pascal}}Id,
	}
}
