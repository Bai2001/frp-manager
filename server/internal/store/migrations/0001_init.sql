-- FRP Manager Server 初始化 schema
-- 依据设计文档第 14 节

CREATE TABLE IF NOT EXISTS port_allocations (
    id          TEXT PRIMARY KEY,
    protocol    TEXT NOT NULL,
    port        INTEGER NOT NULL,
    tunnel_id   TEXT,
    client_id   TEXT,
    status      TEXT NOT NULL DEFAULT 'allocated',
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL,
    UNIQUE(protocol, port)
);

CREATE TABLE IF NOT EXISTS domain_allocations (
    id          TEXT PRIMARY KEY,
    protocol    TEXT NOT NULL,
    domain      TEXT NOT NULL,
    tunnel_id   TEXT,
    client_id   TEXT,
    status      TEXT NOT NULL DEFAULT 'allocated',
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL,
    UNIQUE(protocol, domain)
);

CREATE TABLE IF NOT EXISTS server_settings (
    key   TEXT PRIMARY KEY,
    value TEXT
);
