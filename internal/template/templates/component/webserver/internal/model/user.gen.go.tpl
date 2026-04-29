// 注意：本文件原本由 gorm.io/gen 生成（参考 miniblog-v4），linctl 脚手架阶段是手写的等价版本。
// 当你引入 `make gen-model` 类的工作流后，可以删除本文件并由代码生成器接管。
//
// 表名与列名与 configs/{init_database,basic}.sql 严格对齐：
// 表名 sys_user（避开 PostgreSQL 关键字 user），列名 snake_case。

package model

import (
	"time"
)

// TableNameUserM 是 UserM 对应的数据表名.
const TableNameUserM = "sys_user"

// UserM 映射数据库 sys_user 表.
//
// 说明：约束（unique / index / 类型）以 configs/init_database.sql 为唯一真相，
// 这里不重复声明 uniqueIndex/index，避免 GORM AutoMigrate 与 SQL 已建索引名冲突。
type UserM struct {
	ID          int64      `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	UserID      string     `gorm:"column:user_id;type:uuid" json:"userID"`
	Username    string     `gorm:"column:username;size:50;not null" json:"username"`
	Password    string     `gorm:"column:password;size:128;not null" json:"-"`
	Nickname    string     `gorm:"column:nickname;size:50" json:"nickname"`
	Email       string     `gorm:"column:email;size:255" json:"email"`
	Phone       string     `gorm:"column:phone;size:20" json:"phone"`
	Avatar      string     `gorm:"column:avatar;size:500" json:"avatar"`
	Gender      int16      `gorm:"column:gender;not null;default:0" json:"gender"`
	Status      int16      `gorm:"column:status;not null;default:0" json:"status"`
	LastLoginAt *time.Time `gorm:"column:last_login_at" json:"lastLoginAt,omitempty"`
	Description string     `gorm:"column:description;type:text" json:"description"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null;default:current_timestamp;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;not null;default:current_timestamp;autoUpdateTime" json:"updatedAt"`
}

// TableName 返回 UserM 对应的数据表名.
func (*UserM) TableName() string {
	return TableNameUserM
}
