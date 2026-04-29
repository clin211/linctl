// Package user 实现 user 资源在业务层的逻辑（biz layer）。
//
// 与该包同级的每个 .go 文件聚焦一个动作：
//   - user.go         接口定义 + 构造函数
//   - create.go       创建用户（绑定默认角色）
//   - get.go          按租户取出当前登录用户
//   - list.go         分页列出用户（并发组装 v1 视图）
//   - listwithbadperformance.go  顺序版本，作为对照
//   - update.go       局部更新（仅修改非 nil 字段）
//   - delete.go       删除用户（同步移除角色绑定）
//   - changepassword.go 修改密码（先比对旧密码再加密新密码）
//   - login.go        登录后签发 JWT
//   - refreshtoken.go 刷新 JWT
package user
