-- FRP Manager Server 初始化 schema
-- 依据设计文档第 14 节

-- 端口分配记录（TCP/UDP 远程端口占用）
CREATE TABLE IF NOT EXISTS port_allocations (
    id          TEXT PRIMARY KEY,
    protocol    TEXT NOT NULL,                   -- tcp | udp
    port        INTEGER NOT NULL,
    tunnel_id   TEXT,
    client_id   TEXT,
    status      TEXT NOT NULL DEFAULT 'allocated', -- allocated | released
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL,

    UNIQUE(protocol, port)
);

-- 域名分配记录（HTTP/HTTPS 域名占用）
CREATE TABLE IF NOT EXISTS domain_allocations (
    id          TEXT PRIMARY KEY,
    protocol    TEXT NOT NULL,                   -- http | https
    domain      TEXT NOT NULL,
    tunnel_id   TEXT,
    client_id   TEXT,
    status      TEXT NOT NULL DEFAULT 'allocated',
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL,

    UNIQUE(protocol, domain)
);

-- 服务端键值配置
CREATE TABLE IF NOT EXISTS server_settings (
    key   TEXT PRIMARY KEY,
    value TEXT
);
