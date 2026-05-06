package conversion

import (
	"{{.Module}}/internal/{{.AppName}}/model"
	v1 "{{.Module}}/pkg/api/{{.AppName}}/v1"
)

// {{.Resource | Pascal}}MToProto 将 {{.Resource | Pascal}}M 模型转换为 proto 中的 {{.Resource | Pascal}}Info。
func {{.Resource | Pascal}}MToProto(m *model.{{.Resource | Pascal}}M) *v1.{{.Resource | Pascal}}Info {
	return &v1.{{.Resource | Pascal}}Info{
		{{.Resource | Pascal}}Id: m.{{.Resource | Pascal}}ID,
		CreatedAt: m.CreatedAt.Unix(),
		UpdatedAt: m.UpdatedAt.Unix(),
	}
}

// Proto{{.Resource | Pascal}}ToM 将 proto 中的 {{.Resource | Pascal}}Info 转换为 {{.Resource | Pascal}}M 模型。
func Proto{{.Resource | Pascal}}ToM(p *v1.{{.Resource | Pascal}}Info) *model.{{.Resource | Pascal}}M {
	return &model.{{.Resource | Pascal}}M{
		{{.Resource | Pascal}}ID: p.{{.Resource | Pascal}}Id,
	}
}
