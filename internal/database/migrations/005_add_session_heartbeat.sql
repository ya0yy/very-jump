-- 添加会话心跳字段
-- Migration: 005_add_session_heartbeat.sql

-- 添加最后心跳时间字段
ALTER TABLE sessions ADD COLUMN last_heartbeat DATETIME;

-- 为现有的active会话设置心跳时间为启动时间
UPDATE sessions SET last_heartbeat = start_time WHERE status = 'active';

-- 创建索引以提高查询性能
CREATE INDEX idx_sessions_status_heartbeat ON sessions(status, last_heartbeat);