package conversion

import (
	"github.com/jinzhu/copier"

	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/model"
	v1 "{{ .Project.Metadata.Module }}/pkg/api/{{ .Component.Name }}/v1"
)

// copyOpt 是 user 模块通用的拷贝选项：忽略零值字段、深拷贝。
//
// TODO(linctl): 当 model.UserM (time.Time) 与 v1.User (timestamppb.Timestamp) 的时间字段
// 需要互转时，请改用 pkg/core.CopyWithConverters 注入 TypeConverter；
// 当前骨架阶段保留最小依赖，时间字段需在调用方手动处理。
var copyOpt = copier.Option{IgnoreEmpty: true, DeepCopy: true}

// UserModelToUserV1 将模型层的 UserM 转换为 Protobuf 层的 v1.User.
func UserModelToUserV1(userModel *model.UserM) *v1.User {
	var protoUser v1.User
	_ = copier.CopyWithOption(&protoUser, userModel, copyOpt)
	return &protoUser
}

// UserV1ToUserModel 将 Protobuf 层的 v1.User 转换为模型层的 UserM.
func UserV1ToUserModel(protoUser *v1.User) *model.UserM {
	var userModel model.UserM
	_ = copier.CopyWithOption(&userModel, protoUser, copyOpt)
	return &userModel
}
