-- FRP Manager Client 初始化 schema
-- 依据设计文档第 15 节

CREATE TABLE IF NOT EXISTS servers (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    host         TEXT NOT NULL,
    frps_port    INTEGER NOT NULL DEFAULT 7000,
    frp_token    TEXT NOT NULL,
    agent_url    TEXT NOT NULL,
    agent_token  TEXT NOT NULL,
    is_default   INTEGER NOT NULL DEFAULT 0,
    remark       TEXT,
    created_at   DATETIME NOT NULL,
    updated_at   DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS tunnels (
    id            TEXT PRIMARY KEY,
    server_id     TEXT NOT NULL,
    name          TEXT NOT NULL,
    protocol      TEXT NOT NULL,           -- tcp | udp | http | https
    local_ip      TEXT NOT NULL DEFAULT '127.0.0.1',
    local_port    INTEGER NOT NULL,
    remote_port   INTEGER,                 -- tcp/udp 使用
    custom_domain TEXT,                    -- http/https 自定义域名模式
    subdomain     TEXT,                    -- http/https 子域名模式
    enabled       INTEGER NOT NULL DEFAULT 1,
    status        TEXT NOT NULL DEFAULT 'stopped',
    last_error    TEXT,
    remark        TEXT,
    created_at    DATETIME NOT NULL,
    updated_at    DATETIME NOT NULL,
    FOREIGN KEY (server_id) REFERENCES servers(id)
);

CREATE INDEX IF NOT EXISTS idx_tunnels_server_id ON tunnels(server_id);
