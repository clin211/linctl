package model

import (
	"gorm.io/gorm"

	"{{ .Project.Metadata.Module }}/internal/pkg/rid"
	"{{ .Project.Metadata.Module }}/pkg/authn"
)

// BeforeCreate 在写入数据库前对明文密码做哈希加密.
func (m *UserM) BeforeCreate(tx *gorm.DB) error {
	var err error
	m.Password, err = authn.Encrypt(m.Password)
	if err != nil {
		return err
	}

	return nil
}

// AfterCreate 在自增 ID 落库后生成业务可见的 userID（"user-xxxxxx"）并回写.
func (m *UserM) AfterCreate(tx *gorm.DB) error {
	m.UserID = rid.UserID.New(uint64(m.ID))

	return tx.Save(m).Error
}

// 注：sys_user 的 schema 由 configs/init_database.sql 创建，不依赖 GORM AutoMigrate。
// 如需启用自动迁移：
//  1. 在 import 中加回 "{{ .Project.Metadata.Module }}/pkg/store/registry"
//  2. 加回 init() { registry.Register(&UserM{}) }
//  3. 确保 model 的 uniqueIndex/index tag 与 SQL 中已建索引的命名保持一致
//     （GORM 默认约定：uni_<table>_<col> / idx_<table>_<col>），否则 AutoMigrate
//     会尝试 DROP 不存在的约束并失败。
