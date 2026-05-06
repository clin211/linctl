//go:generate go run {{.Module}}/cmd/gen-gorm-model
package model

import "time"

// {{.Resource | Pascal}}M 是 {{.Resource | Pascal}} 的数据库映射模型（占位实现）。
//
// ⚠️ 该文件由 lin 生成的占位实现，仅用于让项目在 `linctl add` 之后能立刻编译。推荐流程：
//  1. 创建对应的数据库表（DDL 由开发者手动维护）。
//  2. 运行 `make gen-model` 调用 cmd/gen-gorm-model，根据数据库 schema 自动生成真实 model。
//  3. 占位文件会被覆盖；将 gorm gen 的输出提交至仓库。
type {{.Resource | Pascal}}M struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	{{.Resource | Pascal}}ID string    `gorm:"column:{{.Resource | Snake}}_id;uniqueIndex;type:varchar(35);not null" json:"{{.Resource | LowerCamel}}ID"`
	CreatedAt time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

// TableName 指定 {{.Resource | Pascal}}M 对应的数据库表名。
func ({{.Resource | Pascal}}M) TableName() string {
	return "{{.Resource | Snake | Plural}}"
}
