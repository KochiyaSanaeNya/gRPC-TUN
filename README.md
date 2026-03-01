# gRPC-TUN
借助GPT5.2codex,在原HTTPSH(未公开)上重写而来,gRPC隧道软件,功能较为单一,带TLS加密

# 使用说明

使用 gRPC 对 TCP 数据进行封装与转发的隧道程序，支持自动 TLS（可自签名 + 指纹校验）与可选明文模式。

## 配置

程序由 JSON 配置文件驱动。

### 服务端示例

```json
{
  "mode": "server",
  "server": {
    "listen_addr": "0.0.0.0:8443",
    "target_addr": "127.0.0.1:5432"
  },
  "tls": {
    "cert_file": "server.pem",
    "key_file": "server-key.pem"
  }
}
```

- 如果 `tls.cert_file`/`tls.key_file` 为空，启动时自动生成自签名证书，并打印 SHA256 指纹。
- 客户端可使用该指纹进行证书绑定校验。

### 客户端示例

```json
{
  "mode": "client",
  "client": {
    "local_listen": "127.0.0.1:15432",
    "server_addr": "server.example.com:8443",
    "dial_timeout_seconds": 10
  },
  "tls": {
    "server_name": "server.example.com",
    "pinned_fingerprint_sha256": "<paste server fingerprint here>"
  }
}
```

### 明文模式示例（不启用 TLS）

```json
{
  "mode": "client",
  "client": {
    "local_listen": "127.0.0.1:15432",
    "server_addr": "127.0.0.1:8443",
    "dial_timeout_seconds": 10
  },
  "tls": {
    "allow_plaintext": true
  }
}
```

> 明文模式仅建议在内网或可信网络中使用。

## 运行

```bash
# 服务端
./wsssh -cfg server.json

# 客户端
./wsssh -cfg client.json
```

## 配置字段说明

- `mode`: `client` 或 `server`
- `client.local_listen`: 客户端本地监听地址（TCP）
- `client.server_addr`: 服务端 gRPC 地址
- `client.dial_timeout_seconds`: 连接服务端的超时时间（秒）
- `server.listen_addr`: 服务端监听地址（gRPC）
- `server.target_addr`: 服务端转发目标地址（TCP）
- `tls.cert_file`/`tls.key_file`: 服务端证书与私钥文件（需 `.pem` 后缀）
- `tls.ca_file`: 客户端 CA 文件（可选）
- `tls.server_name`: 客户端 TLS ServerName（可选）
- `tls.pinned_fingerprint_sha256`: 服务端证书指纹（SHA256，可选）
- `tls.insecure_skip_verify`: 跳过证书校验（不推荐）
- `tls.allow_plaintext`: 明文模式开关（不启用 TLS）
