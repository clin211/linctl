--
-- {{ .Project.Metadata.Name }} - 基础数据库结构
-- Modules: 用户管理 + Casbin RBAC + 登录日志 + 系统监控
-- Database: PostgreSQL 16+
-- Description: 与 linctl WebServer + Stream A user 资源对齐的初始 schema
--
-- 注意：
-- 1. 此脚本需要在 {{ snake .Project.Metadata.Name }} 数据库中执行
-- 2. 请先通过 init_database.sql 创建数据库
-- 3. 再使用 `psql -d {{ snake .Project.Metadata.Name }} -f basic.sql` 应用本脚本
--

-- 基础设置
SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = ON;
SELECT pg_catalog.set_config('search_path', 'public', FALSE);
SET check_function_bodies = FALSE;
SET xmloption = CONTENT;
SET client_min_messages = warning;
SET row_security = OFF;

SET default_tablespace = '';
SET default_table_access_method = heap;

-- 启用 UUID 扩展（sys_user.user_id 使用 uuid_generate_v4()）
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 清理现有表（按依赖关系倒序）
DROP TABLE IF EXISTS public.sys_user_config;
DROP TABLE IF EXISTS public.sys_user;
DROP TABLE IF EXISTS public.casbin_rule;
DROP TABLE IF EXISTS public.sys_login_log;
DROP TABLE IF EXISTS public.sys_monitor;

-- =========================
-- 用户管理模块 (User Management)
-- =========================
-- 用户表
-- Description: 系统用户基础信息表，存储用户的基本资料和认证信息
-- 使用说明:
--   - id: 内部自增主键，关联与统计性能最优
--   - user_id (UUID): 对外 API 标识，安全且分布式唯一
--   - password: 存储 bcrypt 加密后的密码，禁止明文
--   - status: 0=正常 1=锁定 2=禁用 3=注销
CREATE TABLE public.sys_user (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    username VARCHAR(50) NOT NULL UNIQUE,
    password VARCHAR(128) NOT NULL,
    email VARCHAR(255) NULL UNIQUE,
    phone VARCHAR(20) NULL UNIQUE,
    avatar VARCHAR(500) NULL,
    gender SMALLINT NOT NULL DEFAULT 0,
    status SMALLINT NOT NULL DEFAULT 0,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    description TEXT
);

COMMENT ON TABLE sys_user IS '用户基础信息表，存储用户认证信息和基本资料';

CREATE INDEX idx_sys_user_status ON sys_user (status);
CREATE INDEX idx_sys_user_status_created ON sys_user (status, created_at DESC);
CREATE INDEX idx_sys_user_status_last_login ON sys_user (status, last_login_at DESC);

-- =========================
-- Casbin 权限控制模块
-- =========================
-- Casbin 规则表
-- Description: 存储 Casbin 权限策略规则。pkg/authz 默认通过 gorm-adapter 读写本表。
-- 使用说明:
--   - 基于「主体, 对象, 动作」模型 (subject, object, action)
--   - ptype: p=权限策略, g=角色继承
--   - 修改后由 enforcer.AutoLoadPolicy 周期性热加载
CREATE TABLE public.casbin_rule (
    id BIGSERIAL PRIMARY KEY,
    ptype VARCHAR(100) NOT NULL,
    v0 VARCHAR(100),
    v1 VARCHAR(100),
    v2 VARCHAR(100),
    v3 VARCHAR(100),
    v4 VARCHAR(100),
    v5 VARCHAR(100)
);

COMMENT ON TABLE casbin_rule IS 'Casbin 权限规则表（gorm-adapter 持久化目标）';

CREATE INDEX idx_casbin_rule_ptype ON casbin_rule (ptype);
CREATE INDEX idx_casbin_rule_ptype_v0_v1 ON casbin_rule (ptype, v0, v1);
CREATE INDEX idx_casbin_rule_ptype_v0_v1_v2 ON casbin_rule (ptype, v0, v1, v2);
CREATE INDEX idx_casbin_rule_g_v0 ON casbin_rule (ptype, v0)
WHERE ptype = 'g';

-- =========================
-- 登录日志（用于安全审计）
-- =========================
CREATE TABLE public.sys_login_log (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50),
    ip_address INET,
    user_agent VARCHAR(1000),
    status BOOLEAN NOT NULL,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE sys_login_log IS '用户登录日志表，记录登录尝试和安全信息';

CREATE INDEX idx_sys_login_log_user_created ON sys_login_log (username, created_at DESC);
CREATE INDEX idx_sys_login_log_status_created ON sys_login_log (status, created_at DESC);
CREATE INDEX idx_sys_login_log_ip_created ON sys_login_log (ip_address, created_at DESC);
CREATE INDEX idx_sys_login_log_user_status ON sys_login_log (username, status);
CREATE INDEX idx_sys_login_log_username ON sys_login_log (username);

-- =========================
-- 用户个人配置
-- =========================
CREATE TABLE public.sys_user_config (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES sys_user (user_id) ON DELETE CASCADE,
    config_key VARCHAR(100) NOT NULL,
    config_value JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, config_key)
);

COMMENT ON TABLE sys_user_config IS '用户个人配置表，存储用户偏好设置（JSONB）';

-- =========================
-- 系统监控
-- =========================
CREATE TABLE public.sys_monitor (
    id BIGSERIAL PRIMARY KEY,
    server_name VARCHAR(100),
    server_ip INET,
    cpu_usage DECIMAL(5, 2) NOT NULL,
    cpu_cores INTEGER,
    memory_usage DECIMAL(5, 2) NOT NULL,
    memory_total BIGINT,
    memory_used BIGINT,
    disk_usage DECIMAL(5, 2) NOT NULL,
    disk_total BIGINT,
    disk_used BIGINT,
    network_rx BIGINT DEFAULT 0,
    network_tx BIGINT DEFAULT 0,
    load_avg_1 DECIMAL(5, 2),
    uptime BIGINT,
    process_count INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE sys_monitor IS '系统监控表，记录服务器资源使用情况';

CREATE INDEX idx_sys_monitor_server_created ON sys_monitor (server_ip, created_at DESC);
CREATE INDEX idx_sys_monitor_created_at ON sys_monitor (created_at DESC);
CREATE INDEX idx_sys_monitor_high_cpu ON sys_monitor (created_at DESC)
WHERE cpu_usage > 80.0;
CREATE INDEX idx_sys_monitor_high_memory ON sys_monitor (created_at DESC)
WHERE memory_usage > 80.0;

-- =========================
-- 初始数据 (Initial Data)
-- =========================
-- 默认超级管理员账户（用户名 admin / 密码 admin123；生产环境务必立即修改）
INSERT INTO sys_user (
    username,
    password,
    email,
    phone,
    description
) VALUES (
    'admin',
    -- bcrypt hash of "admin123"（与 miniblog-v4 注释保持一致；上一版 hash 实际是 "password" 的 hash）
    '$2a$10$Ntu0XC0bzIOG7kUBcKkEl.4llVMqA8XbIm9r.1XZzDwPFuYr1u6qG',
    'admin@{{ snake .Project.Metadata.Name }}.local',
    '+8613800138000',
    '系统默认超级管理员账户，由 linctl 初始化创建'
);

-- =========================
-- Casbin 初始策略
-- =========================
-- 用户角色绑定 (g 策略)
INSERT INTO casbin_rule (ptype, v0, v1, v2) VALUES
    ('g', 'admin', 'r:super_admin', NULL);

-- 角色权限策略 (p 策略)；与 configs/casbin/model.conf 中 p = sub, obj, act, eft 对齐，
-- v3 必须是 'allow' 或 'deny'，否则 NewServer 时 Casbin 会拒绝加载。
INSERT INTO casbin_rule (ptype, v0, v1, v2, v3) VALUES
    -- 超级管理员可访问 /api/v1/* 全部方法
    ('p', 'r:super_admin', '/api/v1/*', 'GET', 'allow'),
    ('p', 'r:super_admin', '/api/v1/*', 'POST', 'allow'),
    ('p', 'r:super_admin', '/api/v1/*', 'PUT', 'allow'),
    ('p', 'r:super_admin', '/api/v1/*', 'DELETE', 'allow'),
    -- 系统监控（管理员 / 普通用户均可读）
    ('p', 'r:admin', '/api/v1/monitor/*', 'GET', 'allow'),
    ('p', 'r:user', '/api/v1/monitor/*', 'GET', 'allow'),
    -- Casbin 规则管理（仅超级管理员）
    ('p', 'r:super_admin', '/api/v1/casbin/rules', 'GET', 'allow'),
    ('p', 'r:super_admin', '/api/v1/casbin/rules', 'POST', 'allow'),
    ('p', 'r:super_admin', '/api/v1/casbin/rules/*', 'PUT', 'allow'),
    ('p', 'r:super_admin', '/api/v1/casbin/rules/*', 'DELETE', 'allow'),
    ('p', 'r:super_admin', '/api/v1/casbin/sync', 'POST', 'allow'),
    -- 日志查看
    ('p', 'r:admin', '/api/v1/logs/*', 'GET', 'allow'),
    ('p', 'r:user', '/api/v1/logs/login', 'GET', 'allow'),
    -- 用户个人 profile（任意已登录用户）
    ('p', '*', '/api/v1/users/profile', 'GET', 'allow'),
    ('p', '*', '/api/v1/users/profile', 'PUT', 'allow');
