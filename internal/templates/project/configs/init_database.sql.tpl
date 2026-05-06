-- {{.ProjectName | Title}} - 数据库初始化脚本
-- 描述：创建 {{.AppName}} 数据库并设置基础配置
-- 用法：先执行此脚本创建数据库，再执行 basic.sql 创建表结构

{{- if eq .Storage "gorm-postgres" }}
-- 创建 {{.AppName}} 数据库
DROP DATABASE IF EXISTS {{.AppName}};
CREATE DATABASE {{.AppName}}
    WITH TEMPLATE = template0
    ENCODING = 'UTF8'
    LOCALE_PROVIDER = libc
    LOCALE = 'en_US.utf8';

-- 添加数据库注释
COMMENT ON DATABASE {{.AppName}} IS '{{.ProjectName | Title}} 服务数据库。';

-- 设置数据库所有者
ALTER DATABASE {{.AppName}} OWNER TO postgres;

-- 输出成功信息
DO $$
BEGIN
    RAISE NOTICE '数据库 {{.AppName}} 创建成功！';
    RAISE NOTICE '接下来请执行：psql -d {{.AppName}} -f basic.sql';
END $$;
{{- else if eq .Storage "gorm-mysql" }}
-- 创建 {{.AppName}} 数据库
DROP DATABASE IF EXISTS {{.AppName}};
CREATE DATABASE {{.AppName}}
    DEFAULT CHARACTER SET utf8mb4
    DEFAULT COLLATE utf8mb4_general_ci;

-- 输出提示
SELECT '数据库 {{.AppName}} 创建成功！' AS message;
SELECT '接下来请执行：mysql -u root -p {{.AppName}} < basic.sql' AS next_step;
{{- else if eq .Storage "gorm-sqlite" }}
-- SQLite 在打开数据库文件时自动创建，无需初始化脚本。
-- 如需建表，请直接执行 basic.sql。
{{- else }}
-- 当前存储后端为 {{.Storage}}，无需 SQL 初始化。
{{- end }}
