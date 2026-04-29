package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"{{ .Project.Metadata.Module }}/pkg/db"
	"github.com/samber/lo"
	"github.com/spf13/pflag"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

// 帮助信息文本.
const helpText = `Usage: main [flags] arg [arg...]

This is a pflag example.

Flags:
`

// Querier 定义了数据库查询接口.
type Querier interface {
	// FilterWithNameAndRole 按名称和角色查询记录
	FilterWithNameAndRole(name string) ([]gen.T, error)
}

// GenerateConfig 保存代码生成的配置.
type GenerateConfig struct {
	ModelPackagePath string
	GenerateFunc     func(g *gen.Generator)
}

// 预定义的生成配置.
var generateConfigs = map[string]GenerateConfig{
	"mb": {ModelPackagePath: "../../internal/{{ .Component.Name }}/model", GenerateFunc: GenerateModels},
}

// 命令行参数.
var (
	addr       = "localhost:54321"
	username   = "postgres"
	password   = "postgres"
	database   = "{{ .Project.Metadata.Name }}"
	modelPath  = ""
	components = pflag.StringSlice("component", []string{"mb"}, "Generated model code's for specified component.")
	help       = pflag.BoolP("help", "h", false, "Show this help message.")
)

func main() {
	// 设置自定义的使用说明函数
	pflag.Usage = func() {
		fmt.Printf("%s", helpText)
		pflag.PrintDefaults()
	}
	pflag.Parse()

	// 如果设置了帮助标志，则显示帮助信息并退出
	if *help {
		pflag.Usage()
		return
	}

	// 初始化数据库连接
	dbInstance, err := initializeDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 处理组件并生成代码
	for _, component := range *components {
		processComponent(component, dbInstance)
	}
}

// initializeDatabase 创建并返回一个数据库连接.
func initializeDatabase() (*gorm.DB, error) {
	dbOptions := &db.PostgreSQLOptions{
		Addr:     addr,
		Username: username,
		Password: password,
		Database: database,
	}

	// 创建并返回数据库连接
	return db.NewPostgreSQL(dbOptions)
}

// processComponent 处理单个组件以生成代码.
func processComponent(component string, dbInstance *gorm.DB) {
	config, ok := generateConfigs[component]
	if !ok {
		log.Printf("Component '%s' not found in configuration. Skipping.", component)
		return
	}

	// 解析模型包路径
	modelPkgPath := resolveModelPackagePath(config.ModelPackagePath)

	// 创建生成器实例
	generator := createGenerator(modelPkgPath)
	generator.UseDB(dbInstance)

	// 应用自定义生成器选项
	applyGeneratorOptions(generator)

	// 使用指定的函数生成模型
	config.GenerateFunc(generator)

	// 执行代码生成
	generator.Execute()
}

// resolveModelPackagePath 确定模型生成的包路径.
func resolveModelPackagePath(defaultPath string) string {
	if modelPath != "" {
		return modelPath
	}
	absPath, err := filepath.Abs(defaultPath)
	if err != nil {
		log.Printf("Error resolving path: %v", err)
		return defaultPath
	}
	return absPath
}

// createGenerator 初始化并返回一个新的生成器实例.
func createGenerator(packagePath string) *gen.Generator {
	return gen.NewGenerator(gen.Config{
		Mode:              gen.WithDefaultQuery | gen.WithQueryInterface | gen.WithoutContext,
		ModelPkgPath:      packagePath,
		WithUnitTest:      true,
		FieldNullable:     true,  // 对于数据库中可空的字段，使用指针类型。
		FieldSignable:     false, // 禁用无符号属性以提高兼容性。
		FieldWithIndexTag: false, // 不包含 GORM 的索引标签。
		FieldWithTypeTag:  false, // 不包含 GORM 的类型标签。
	})
}

// applyGeneratorOptions 设置自定义生成器选项.
func applyGeneratorOptions(g *gen.Generator) {
	// 为特定字段自定义 GORM 标签
	g.WithOpts(
		// 将下划线连接起来的单词使用小驼峰的形式写入 json tag 中
		gen.FieldJSONTagWithNS(func(columnName string) (tagContent string) {
			// 只对包含下划线的字段名进行转换
			if strings.Contains(columnName, "_") {
				return lo.CamelCase(columnName)
			}
			return columnName
		}),
		// 为时间戳字段设置 PostgreSQL 默认值
		gen.FieldGORMTag("created_at", func(tag field.GormTag) field.GormTag {
			tag.Set("default", "current_timestamp")
			return tag
		}),
		gen.FieldGORMTag("updated_at", func(tag field.GormTag) field.GormTag {
			tag.Set("default", "current_timestamp")
			return tag
		}),
	)
}

// GenerateModels 为应用组件生成模型.
//
// 提示：根据你实际数据库 schema 调整下面的表名 → 模型名映射。
// 这里的示例对应 miniblog-v4 风格的 RBAC + 用户表，可以按需增删。
func GenerateModels(g *gen.Generator) {
	g.GenerateModelAs("sys_api", "ApiM")
	g.GenerateModelAs("sys_button", "ButtonM")
	g.GenerateModelAs("sys_menu", "MenuM")
	g.GenerateModelAs("sys_role", "RoleM")
	g.GenerateModelAs("sys_role_apis", "RoleApisM")
	g.GenerateModelAs("sys_role_buttons", "RoleButtonM")
	g.GenerateModelAs("sys_user_roles", "UserRoleM")
	g.GenerateModelAs("sys_users", "UsersM")
}
