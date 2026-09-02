# 数据存储与加密

vShell 的全部持久化都在本机：一个 SQLite 数据库 + 一个加密密钥文件。没有云端、没有遥测。

---

## 1. 文件位置（macOS）

| 文件 | 路径 | 内容 |
|------|------|------|
| 数据库 | `~/Library/Application Support/vshell/vshell.db` | 连接 / 分组 / 快捷命令 / 端口转发 |
| 加密密钥 | `~/Library/Application Support/vshell/.enc_key` | base64 的 32 字节 AES 密钥（目录 0700、文件 0600） |

路径由 `os.UserConfigDir()` 推导，Linux 下为 `~/.config/vshell/`，Windows 为 `%AppData%\vshell\`。

## 2. SQLite 配置

| 项 | 值 | 说明 |
|----|----|------|
| 驱动 | `modernc.org/sqlite` | 纯 Go，**无 CGO**（红线 6） |
| 日志模式 | WAL | DSN `_pragma=journal_mode(WAL)` |
| 外键 | 开启 | `_pragma=foreign_keys(1)` |
| 连接数 | `SetMaxOpenConns(1)` | 单连接串行化，规避 SQLite 锁问题 |

`db.DB` 为 `sync.Once` 单例，内嵌 `*sql.DB` 并持有 `CryptoService`。应用退出时统一 `Close`。

## 3. 表结构

### groups

| 列 | 类型 | 约束 |
|----|------|------|
| id | TEXT | PRIMARY KEY |
| name | TEXT | NOT NULL |
| parent_id | TEXT | REFERENCES groups(id) ON DELETE **CASCADE** |
| sort_order | INTEGER | DEFAULT 0 |
| created_at / updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP |

### connections

| 列 | 类型 | 约束 / 说明 |
|----|------|-------------|
| id | TEXT | PRIMARY KEY |
| group_id | TEXT | REFERENCES groups(id) ON DELETE **SET NULL** |
| name / host / username | TEXT | NOT NULL |
| port | INTEGER | DEFAULT 22 |
| auth_type | TEXT | NOT NULL（password / private_key / agent / interactive） |
| password / private_key / key_passphrase | TEXT | **AES-256-GCM 密文** |
| key_name | TEXT | 托管密钥名（相对 `~/.ssh`） |
| proxy_type / proxy_addr / jump_host_id | TEXT | 预留，未启用 |
| upload_path | TEXT | DEFAULT '/' |
| default_cmd | TEXT | 预留 |
| sort_order / color / last_used_at | — | color 等字段暂无 UI |
| created_at / updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP |

### quick_commands / port_forwards

`quick_commands(id, name, command NOT NULL, connection_id, sort_order, …)` 与 `port_forwards(id, name, connection_id NOT NULL, type NOT NULL, local_host, local_port, remote_host, remote_port, auto_start DEFAULT 0)`。两者的 `connection_id` **未设外键**（手工删除连接记录不会级联）。

### 迁移机制

无版本表：每次启动重放 `CREATE TABLE IF NOT EXISTS` + 容错的 `ALTER TABLE ADD COLUMN`（列已存在时忽略错误）。适合当前规模；引入破坏性变更前需要升级为版本化迁移。

## 4. AES-256-GCM 凭证加密（`internal/crypto`）

### 4.1 密钥派生

按优先级：

1. 环境变量 `VSHELL_ENCRYPTION_KEY`（base64，须 32 字节）——便于备份 / 迁移场景
2. 已存在的 `.enc_key` 文件
3. 首次运行用 `crypto/rand` 生成 32 字节随机密钥写入 `.enc_key`

> 注意：密钥**不是**机器绑定，也不入 Keychain——就是配置目录里的文件。备份整个 `vshell/` 目录（含 `.enc_key`）即可在其他机器解密；只拷数据库则无法解密。

### 4.2 加解密流程

```text
Encrypt(plain)                          Decrypt(cipher)
  nonce = rand(12)                         raw   = base64decode(cipher)
  ct    = AES-256-GCM.Seal(nonce,          nonce = raw[:12]
            key, nonce, plain)             pt    = GCM.Open(key, nonce,
  return base64(nonce ‖ ct)                       raw[12:])   // 认证校验失败即报错
```

- 空字符串直接存空（不加密）
- GCM 自带完整性认证，被篡改的密文解密时报错

### 4.3 哪些字段加密

仅 `connections` 表的 `password` / `private_key` / `key_passphrase` 三列。分组名、命令、端口转发等均为明文（不含凭证）。

### 4.4 读写路径

| 场景 | 流程 |
|------|------|
| 保存连接 | `CreateConnection` / `UpdateConnection` 先 `Encrypt` 再入库；更新时敏感字段留空则保留旧密文 |
| 列表 | `ListConnections` SELECT 显式排除三列，**密文不出后端** |
| 表单回显 | `GetPassword` / `GetPrivateKey` / `GetKeyPassphrase` 解密返回 |
| 建立连接 | `buildSSHConfig` 取全列解密后用于认证 |
| 密钥删除保护 | `GetSSHKeyUsage` 除按 `key_name` 匹配外，解密各连接私钥逐字节比对内容 |

## 5. 数据安全边界（如实说明）

- SSH `HostKeyCallback` 当前为 `InsecureIgnoreHostKey()`，不校验主机指纹（无 known_hosts 集成）
- `.enc_key` 与数据库同目录，防不了本机已被入侵的攻击者，防的是数据库文件单独泄露
- 敏感字段明文仅存在于后端内存与加密前的表单提交中

---

## 延伸阅读

- [架构总览](architecture.md)
- [连接与分组管理](../guide/connections.md)
