# Backend API, Production Hardening & Frontend Integration — 技术设计方案

**版本**: v1.0  
**日期**: 2026-07-10  
**状态**: Draft  
**依赖**: VideoStreamCaptureEngine Phase 1 + 1.5（已完成）

---

## 1. 概述

在已有引擎基础上增加三层能力：

| 层 | 目标 |
|---|------|
| 后端 API | 17 个 REST 端点，覆盖流管理、事件、配置、认证、实时数据 |
| 生产加固 | JWT 认证 + bcrypt 密码哈希 + 环境变量覆盖敏感配置 + PostgreSQL 持久化 |
| 前端对接 | 删除所有 mock 数据，前端替换为真实 API 调用 |

---

## 2. 后端 API 设计

### 2.1 端点清单

**认证**
| 方法 | 路径 | 请求体 | 响应 |
|------|------|--------|------|
| POST | `/api/login` | `{username, password}` | `{token, expires_at}` |
| POST | `/api/refresh` | Bearer token | `{token, expires_at}` |

**流管理**
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/streams` | 列表：`?page=1&size=20&status=running&group=园区-北门&q=gate` |
| GET | `/api/streams/:id` | 详情：含实时 FPS、延迟、出帧数、丢帧率、配置、过滤链 |
| POST | `/api/streams` | 创建流（JSON body = StreamConfig 字段） |
| PUT | `/api/streams/:id` | 更新流配置 |
| DELETE | `/api/streams/:id` | 删除流（需先停止） |
| POST | `/api/streams/:id/start` | 启动流 |
| POST | `/api/streams/:id/stop` | 停止流 |
| POST | `/api/streams/:id/restart` | 重启流 |
| POST | `/api/streams/batch` | 批量导入：multipart YAML/JSON 文件或 JSON body |

**事件**
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/events` | 列表：`?page=1&size=20&level=error&stream_id=gate-north&from=&to=&q=` |
| POST | `/api/events/ack` | 批量确认：`{ids: [1,2,3]}` 或 `{all: true}` |

**配置**
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/config` | 获取全局配置（引擎参数、Kafka、序列化、数据库、重启策略） |
| PUT | `/api/config` | 更新全局配置（部分参数热更新，部分需重启） |

**实时数据**
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/metrics/summary` | 仪表盘汇总：在线数/总数、今日出帧、平均FPS、活跃告警、FPS 趋势数组 |
| GET | `/api/streams/:id/frame` | 获取最新 JPEG 帧（base64 编码的 data URL 或 binary） |

### 2.2 统一响应格式

```json
{
  "code": 0,
  "data": { ... },
  "message": "ok"
}
```

| code | 含义 |
|------|------|
| 0 | 成功 |
| 401 | 未认证 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 409 | 冲突（如重复启动） |
| 500 | 服务端错误 |

### 2.3 流管理 API 响应示例

```json
// GET /api/streams
{
  "code": 0,
  "data": {
    "total": 50,
    "page": 1,
    "size": 20,
    "items": [
      {
        "id": "gate-north",
        "group": "园区-北门",
        "status": "running",
        "fps": 25.0,
        "resolution": "1920x1080",
        "frames_total": 2147833,
        "latency_ms": 42,
        "uptime": "3d 14h",
        "rtsp_url": "rtsp://10.0.2.101/stream1",
        "output_topic": "gate-north",
        "capture_fps": 25,
        "decode_scale": "1920x1080",
        "filters": [{"type":"duplicate","params":{"threshold":10}}],
        "restart": {"max_retries":20,"backoff_initial":"1s","backoff_max":"60s","backoff_factor":2.0}
      }
    ]
  }
}
```

---

## 3. 安全加固

### 3.1 JWT 认证

- 算法：HMAC-SHA256，密钥从环境变量 `JWT_SECRET` 读取
- 过期时间：24 小时
- 中间件：解析 `Authorization: Bearer <token>`，验证签名和过期时间
- 非 `/api/login`、`/api/refresh`、`/health`、`/metrics` 的所有 `/api/*` 路径均需认证

### 3.2 密码管理

- 引擎首次启动：从 `ADMIN_PASSWORD` 环境变量读取初始密码
- bcrypt cost=12 哈希后存入 PostgreSQL `users` 表
- 登录验证：`SELECT password_hash FROM users WHERE username=$1` → `bcrypt.Compare`
- 支持通过 API 修改密码（需旧密码验证）

### 3.3 环境变量覆盖

配置加载优先级：**环境变量 > YAML 文件 > 默认值**

| 环境变量 | 覆盖字段 |
|----------|---------|
| `DB_HOST` | `database.host` |
| `DB_PORT` | `database.port` |
| `DB_USER` | `database.user` |
| `DB_PASSWORD` | `database.password` |
| `DB_NAME` | `database.dbname` |
| `JWT_SECRET` | JWT 签名密钥 |
| `ADMIN_PASSWORD` | 初始管理员密码 |
| `KAFKA_BROKERS` | `output.kafka.brokers`（逗号分隔） |

### 3.4 PostgreSQL 表结构

```sql
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(64) UNIQUE NOT NULL,
    password_hash VARCHAR(256) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    stream_id VARCHAR(64) NOT NULL DEFAULT '',
    level VARCHAR(16) NOT NULL DEFAULT 'info',
    message TEXT NOT NULL DEFAULT '',
    acknowledged BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_events_stream ON events(stream_id);
CREATE INDEX idx_events_level ON events(level);
CREATE INDEX idx_events_created ON events(created_at DESC);
```

---

## 4. 前端对接

### 4.1 新增文件

```
web/src/
├── api/
│   ├── client.ts         # fetch 封装：baseURL、JWT 注入、统一错误处理
│   ├── auth.ts           # login(), refresh()
│   ├── streams.ts        # listStreams(), getStream(), createStream(), start/stop/restart, batchImport
│   ├── events.ts         # listEvents(), ackEvents()
│   └── config.ts         # getConfig(), updateConfig()
├── hooks/
│   └── useAPI.ts         # 通用数据获取 hook（loading/error/refetch）
```

### 4.2 API Client 封装

```typescript
// api/client.ts
const BASE = '/api';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem('token');
  const res = await fetch(`${BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });
  if (res.status === 401) {
    localStorage.removeItem('token');
    window.location.href = '/login';
  }
  const json = await res.json();
  if (json.code !== 0) throw new Error(json.message);
  return json.data;
}
```

### 4.3 各页面改动

| 页面 | 当前 | 改为 |
|------|------|------|
| `Login.tsx` | `if (username==='admin' && password==='admin')` | `POST /api/login` → 存 token → navigate |
| `Dashboard.tsx` | 3 个硬编码数组 mock | `useAPI('/metrics/summary')` → 渲染 |
| `StreamList.tsx` | `const mockStreams = [...]` | `useAPI('/streams?page=&status=')` → 分页/搜索/筛选 |
| `StreamDetail.tsx` | 全部常量 | `useAPI('/streams/:id')` + 帧预览轮询 |
| `EngineConfig.tsx` | 表单默认值硬编码 | `GET /api/config` → 填充，`PUT /api/config` → 保存 |
| `EventLog.tsx` | `const events = [...]` | `useAPI('/events?level=&stream=')` → 分页/确认 |
| `Layout.tsx` | 顶部统计 "42/50 在线" 硬编码 | 从 Dashboard 数据或独立 API 获取 |

### 4.4 流详情帧预览

```typescript
// 轮询获取最新帧
useEffect(() => {
  const interval = setInterval(async () => {
    const frameUrl = await fetch(`/api/streams/${id}/frame`);
    setFrameSrc(frameUrl);
  }, 2000);
  return () => clearInterval(interval);
}, [id]);
```

---

## 5. 新增/修改文件总览

```
新增:
internal/
├── api/
│   ├── router.go            # ~80 行：路由注册 + JWT 中间件
│   ├── handler_auth.go      # ~60 行：login/refresh
│   ├── handler_stream.go    # ~200 行：流 CRUD + 启停 + 批量导入
│   ├── handler_event.go     # ~80 行：事件查询 + 确认
│   ├── handler_config.go    # ~60 行：配置读写
│   └── handler_metrics.go   # ~50 行：汇总指标 + 帧获取
├── auth/
│   ├── jwt.go               # ~40 行：签发/验证
│   └── password.go          # ~30 行：bcrypt 哈希/比较
├── store/
│   ├── db.go                # ~30 行：pgxpool 连接
│   ├── user_store.go        # ~40 行：用户查询/创建/更新密码
│   └── event_store.go       # ~60 行：事件写入/查询/确认
web/src/
├── api/
│   ├── client.ts            # ~40 行
│   ├── auth.ts              # ~20 行
│   ├── streams.ts           # ~50 行
│   ├── events.ts            # ~20 行
│   └── config.ts            # ~15 行
├── hooks/
│   └── useAPI.ts            # ~40 行

修改:
cmd/engine/main.go           # ~20 行：注册 API 路由 + 启动 store
internal/config/config.go    # ~30 行：EnvOr 函数
web/src/pages/               # 全部 6 页面：替换 mock
web/src/components/Layout.tsx # 顶部统计动态化
go.mod                       # +pgx, golang.org/x/crypto, golang-jwt
```

---

## 6. Go 依赖新增

| 用途 | 库 |
|------|-----|
| PostgreSQL 驱动 | `github.com/jackc/pgx/v5` |
| 连接池 | `github.com/jackc/pgx/v5/pgxpool` |
| bcrypt | `golang.org/x/crypto/bcrypt` |
| JWT | `github.com/golang-jwt/jwt/v5` |

---

## 7. 风险与对策

| 风险 | 对策 |
|------|------|
| JWT 密钥泄露 | 环境变量注入，不写 YAML，生产环境用 K8s Secret |
| 流配置与运行时状态不同步 | 启停操作直接调用 Manager，不通过文件中间层 |
| 批量导入大量流冲击系统 | 复用 `max_concurrent_starts` 限制并发启动 |
| 前端帧预览轮询压力 | 每流 2s 一次，带 ETag/If-None-Match 缓存 |
