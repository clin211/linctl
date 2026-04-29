-- {{ .Project.Metadata.Name }} - 数据库初始化脚本
-- Description: 创建 {{ snake .Project.Metadata.Name }} 数据库并设置基础配置
-- Usage: 先执行此脚本创建数据库，再切换到该库执行 basic.sql 创建表结构
--
-- 执行示例：
--   psql -U postgres -h 127.0.0.1 -p 5432 -f init_database.sql
--   psql -U postgres -h 127.0.0.1 -p 5432 -d {{ snake .Project.Metadata.Name }} -f basic.sql

DROP DATABASE IF EXISTS {{ snake .Project.Metadata.Name }};

CREATE DATABASE {{ snake .Project.Metadata.Name }}
    WITH TEMPLATE = template0
    ENCODING = 'UTF8'
    LOCALE_PROVIDER = libc
    LOCALE = 'en_US.utf8';

COMMENT ON DATABASE {{ snake .Project.Metadata.Name }} IS '{{ .Project.Metadata.Name }} - linctl 生成的应用数据库（含用户管理 / Casbin / 登录日志 / 系统监控等基础模块）';

ALTER DATABASE {{ snake .Project.Metadata.Name }} OWNER TO postgres;

DO $$
BEGIN
    RAISE NOTICE '数据库 {{ snake .Project.Metadata.Name }} 创建成功！';
    RAISE NOTICE '接下来请执行：psql -d {{ snake .Project.Metadata.Name }} -f basic.sql';
END $$;
