# VideoStreamCaptureEngine — 功能需求规格说明书

**版本**: v1.0  
**日期**: 2026-07-10  
**状态**: Draft  

---

## 1. 项目概述

### 1.1 定位

VideoStreamCaptureEngine 是一个**多路 RTSP 视频流帧采集引擎**，核心职责：

> 接收多路 RTSP 视频流 → 解码 → 抽取图像帧 → 可选过滤 → 发布到 Kafka

引擎只做可靠、高效的帧传输，不做业务逻辑。在采集与发布之间预留**可插拔帧过滤管线**，后续可接入轻量 ML 模型过滤低质量帧和重复帧。

### 1.2 关键约束

| 维度 | Phase 1 目标 | Phase 2 扩展 |
|------|-------------|-------------|
| 流数量 | 10-50 路 | 同左 |
| 分辨率 | ≤1080p | 2K / 4K |
| 帧率 | 最高 25fps | 同左 |
| 编码格式 | H264 为主 | H265 支持 |
| 解码方式 | 纯 CPU（ffmpeg 软解） | GPU 硬件加速 |
| 传输格式 | rawvideo pipe | JPEG pipe / 共享内存 |
| 语言 | Go | 同左 |
| 消息队列 | Kafka | 可扩展 |

---

## 2. 整体架构

```
┌────────────────────────────────────────────────────────────────┐
│                    VideoStreamCaptureEngine                     │
│                                                                 │
│  ┌──────────┐   ┌──────────────┐   ┌───────────┐   ┌────────┐ │
│  │ Stream   │   │   Decoder    │   │  Filter   │   │ Output │ │
│  │ Manager  │──→│   Worker     │──→│  Pipeline │──→│ Writer │ │
│  │          │   │  (goroutine) │   │(pluggable)│   │ (Kafka)│ │
│  └──────────┘   └──────────────┘   └───────────┘   └────────┘ │
│       │               │                  │              │      │
│       │          ┌────┴────┐        ┌────┴────┐         │      │
│       │          │ ffmpeg  │        │ Noop /   │         │      │
│       │          │ subproc │        │ DupDet   │         │      │
│       │          │ (pipe)  │        │ (future) │         │      │
│       │          └─────────┘        └─────────┘         │      │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Config Manager │ Health Check │ Metrics / Prometheus    │   │
│  └─────────────────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────────────────┘
```

### 2.1 组件职责

| 组件 | 职责 | Phase 1 | Phase 2 |
|------|------|---------|---------|
| **Stream Manager** | 流生命周期管理（启动/停止/重启）、热更新、优雅关闭 | 完整实现 | — |
| **Decoder Worker** | 管理单个 ffmpeg 子进程，从 pipe 读取帧 | rawvideo only | 多格式 FrameReader |
| **Filter Pipeline** | 可插拔帧过滤链 | Noop + DuplicateFilter | ML 模型 Filter |
| **Output Writer** | 序列化帧并发布到 Kafka | JPEG 序列化 + Kafka | 大帧外部存储、多后端 |
| **Config Manager** | YAML 配置加载与热更新 | 完整实现 | 新增字段忽略即可 |
| **Health Check** | 每路流心跳检测 | 完整实现 | — |
| **Metrics** | Prometheus 指标暴露 | 完整实现 | — |

---

## 3. 组件详细设计

### 3.1 Stream Manager（流管理器）

**职责**：引擎控制平面，管理所有流的生命周期。本身不做编解码，只做调度和状态管理。

#### 功能清单

| 功能 | 描述 |
|------|------|
| 流注册 | 从 YAML 配置文件加载流定义，建立 `stream_id → StreamConfig` 映射 |
| 并发启动 | 按配置并发启动 Decoder Worker，`max_concurrent_starts` 控制同时启动数上限 |
| 热更新 | 监听配置文件变更（fsnotify），对比新旧配置，增量启停 Decoder |
| 优雅关闭 | SIGTERM 时逐路发送停止信号，等待处理后退出（带超时强制 kill） |
| 健康监控 | 定时检查每路 Decoder 的最后出帧时间，超时标记 unhealthy 并触发告警 |
| 重启策略 | 可配置的自动重连：最大重试次数、指数退避间隔 |

#### 数据结构

```go
type StreamConfig struct {
    ID          string        `yaml:"id"`
    RTSPURL     string        `yaml:"rtsp_url"`
    OutputTopic string        `yaml:"output_topic"`
    CaptureFPS  int           `yaml:"capture_fps"`
    DecodeScale string        `yaml:"decode_scale"`
    Filters     []FilterSpec  `yaml:"filters"`
    Restart     RestartPolicy `yaml:"restart"`
    // Phase 2 预留字段（Phase 1 忽略）
    PipeFormat  string        `yaml:"pipe_format"`   // rawvideo | jpeg
    JPEGQuality int           `yaml:"jpeg_quality"`
    HWAccel     string        `yaml:"hwaccel"`        // cuda | vaapi | none
}

type RestartPolicy struct {
    MaxRetries     int           `yaml:"max_retries"`
    BackoffInitial time.Duration `yaml:"backoff_initial"`
    BackoffMax     time.Duration `yaml:"backoff_max"`
    BackoffFactor  float64       `yaml:"backoff_factor"`
}

type StreamState struct {
    Status       string    // running | stopped | unhealthy
    FramesTotal  int64
    LastFrameAt  time.Time
    ErrorsTotal  int64
    StartedAt    time.Time
}
```

#### 并发模型

- Stream Manager：1 个主 goroutine
- 每路流：1 个 Decoder Worker goroutine
- 使用 `errgroup` 管理所有 worker 生命周期
- 热更新通过 `context.CancelFunc` 精确控制单路启停

---

### 3.2 Decoder Worker（解码工作协程）

**职责**：管理单个 ffmpeg 子进程，从 stdout pipe 读取原始帧数据，解码为内存图像，送入 Filter Pipeline。

#### 解码流程

```
┌──────────────────────────────────────────────────────┐
│                  Decoder Worker                        │
│                                                       │
│  ┌──────────┐    pipe     ┌──────────────┐           │
│  │ ffmpeg   │──stdout───→│ FrameReader  │           │
│  │ subproc  │             │ (rawvideo)   │           │
│  │          │←─stderr────│              │           │
│  └──────────┘   (日志)    └──────┬───────┘           │
│                                  │                    │
│                                  ▼                    │
│                          ┌──────────────┐            │
│                          │   Frame      │            │
│                          │   struct{    │            │
│                          │    StreamID  │            │
│                          │    Timestamp │            │
│                          │    Image     │            │
│                          │    SeqNum    │            │
│                          │   }          │            │
│                          └──────┬───────┘            │
│                                 │                    │
└─────────────────────────────────┼────────────────────┘
                                  ▼
                           Filter Pipeline
```

#### ffmpeg 命令（Phase 1）

```bash
ffmpeg \
  -rtsp_transport tcp \
  -rtsp_flags prefer_tcp \
  -stimeout 5000000 \
  -i "rtsp://camera-ip/stream" \
  -f rawvideo \
  -pix_fmt bgr24 \
  -vf "fps=25,scale=1920:1080" \
  pipe:1
```

参数说明：
- `-rtsp_transport tcp`：TCP 传输比 UDP 更可靠，丢包少
- `-stimeout 5000000`：socket 超时 5 秒
- `-f rawvideo`：无容器裸像素输出
- `-pix_fmt bgr24`：BGR 三通道，Go image 库最友好
- `-vf fps=...,scale=...`：帧率控制和分辨率缩放
- `pipe:1`：输出到 stdout

#### FrameReader 接口（Phase 1 定义，Phase 2 扩展）

```go
type FrameReader interface {
    ReadFrame(r io.Reader) (image.Image, error)
}

type RawVideoReader struct {
    frameSize int  // width * height * 3 (BGR)
}

func (r *RawVideoReader) ReadFrame(rd io.Reader) (image.Image, error) {
    buf := make([]byte, r.frameSize)
    if _, err := io.ReadFull(rd, buf); err != nil {
        return nil, err
    }
    // BGR → RGBA 转换
    img := image.NewRGBA(image.Rect(0, 0, r.width, r.height))
    for i := 0; i < len(buf); i += 3 {
        // B, G, R → R, G, B, A
    }
    return img, nil
}
```

Phase 2 新增 `JPEGReader` 时，Decoder Worker 零改动，只改工厂函数。

#### 错误处理与重连

| 场景 | 检测方式 | 处理 |
|------|---------|------|
| RTSP 连接断开 | pipe 读到 EOF | 杀死旧 ffmpeg，退避后重连 |
| ffmpeg 进程崩溃 | `cmd.Wait()` 非零退出 | 同上 + 记录 stderr |
| 网络闪断 | pipe 读异常 | 指数退避：1s → 2s → 4s → ... → max 60s |
| 解码数据异常 | `ReadFrame` 返回 error | 丢弃当前帧，累计计数，连续 N 次报警 |
| 背压（下游慢） | channel 满 | 非阻塞丢弃新帧（保证实时性） |

#### 背压策略

Decoder → buffered channel(cap=5) → Filter Pipeline → Kafka Writer

channel 满时：**丢弃新帧**（non-blocking send），记录丢帧数。优先保证解码侧不阻塞，避免 ffmpeg 内部缓冲区溢出。

---

### 3.3 Filter Pipeline（可插拔过滤链）

**设计目标**：在解码后、Kafka 发布前，插入可串联的帧过滤链。Phase 1 提供接口骨架 + NoopFilter + DuplicateFilter。

#### 接口设计

```go
type FilterDecision int

const (
    FilterPass  FilterDecision = iota  // 放行
    FilterDrop                         // 丢弃
    FilterAbort                        // 丢弃且跳过后续 filter
)

type FrameMeta struct {
    StreamID  string
    SeqNum    int64
    Timestamp time.Time
    ImageSize int
}

type FilteredFrame struct {
    Image image.Image
    Meta  FrameMeta
    Score float64  // 可选质量分数
}

type FrameFilter interface {
    Name() string
    Apply(frame image.Image, meta FrameMeta) (FilteredFrame, FilterDecision)
}
```

#### Pipeline 实现

```go
type FilterPipeline struct {
    filters []FrameFilter
    metrics *FilterMetrics
}

func (p *FilterPipeline) Process(frame image.Image, meta FrameMeta) (*FilteredFrame, bool) {
    current := FilteredFrame{Image: frame, Meta: meta}
    for _, f := range p.filters {
        result, decision := f.Apply(current.Image, current.Meta)
        switch decision {
        case FilterPass:
            current = result
        case FilterDrop, FilterAbort:
            p.metrics.RecordDrop(f.Name(), meta.StreamID)
            return nil, false
        }
    }
    p.metrics.RecordPass(meta.StreamID)
    return &current, true
}
```

#### Phase 1 内置过滤器

**NoopFilter** — 直通，默认过滤器。

**DuplicateFilter** — 基于 dHash（差值哈希）的重复帧检测：
- 将帧缩放到 9×8 灰度图
- 计算相邻像素差值哈希（64 位）
- 与上一帧哈希的汉明距离 ≤ threshold 则丢弃
- 计算耗时约 1ms/帧

#### Phase 2 ML 模型接入

两条路径可选：

1. **进程内推理**：通过 CGO 调用 ONNX Runtime，适合轻量模型（模糊检测、亮度判断）
2. **外部推理服务**：gRPC/HTTP 调用独立 ML 服务，适合重模型（目标检测、场景分类）

Go 端只需实现对应的 `FrameFilter`，Pipeline 逻辑不变。

#### 配置示例

```yaml
streams:
  - id: "camera-01"
    filters:
      - type: "duplicate"
        params:
          threshold: 10
      - type: "noop"
```

---

### 3.4 Output Writer（Kafka 输出）

**职责**：将过滤后的帧序列化为 JPEG，发布到 Kafka。

#### 接口设计

```go
type OutputFrame struct {
    StreamID  string            `json:"stream_id"`
    SeqNum    int64             `json:"seq_num"`
    Timestamp time.Time         `json:"timestamp"`
    Quality   float64           `json:"quality"`     // 过滤器打分
    ImageData []byte            `json:"image_data"`  // JPEG 字节
    ImageSize int               `json:"image_size"`
    Metadata  map[string]string `json:"metadata,omitempty"`
}

type OutputWriter interface {
    Write(frame *OutputFrame) error
    Close() error
    Backend() string
}
```

#### 序列化策略

帧在进 Kafka 前统一编码为 JPEG（质量 85）：

- 1080p JPEG 约 200-400KB，Kafka 默认 1MB 消息限制足够
- Phase 2 大帧（>512KB）可通过外部对象存储（MinIO）存放，Kafka 只传引用 URL

#### Kafka Writer

```go
type KafkaWriter struct {
    producer   sarama.SyncProducer
    serializer *FrameSerializer
    topic      string
}

func (w *KafkaWriter) Write(frame *OutputFrame) error {
    data, _ := json.Marshal(frame)
    _, _, err := w.producer.SendMessage(&sarama.ProducerMessage{
        Topic: w.topic,
        Key:   sarama.StringEncoder(frame.StreamID),  // 同流同分区，保证顺序
        Value: sarama.ByteEncoder(data),
    })
    return err
}
```

#### Phase 2 大帧外部存储策略

```go
type ObjectStorageWriter struct {
    storage   ObjectStorage       // MinIO / S3
    mq        OutputWriter        // Kafka
    threshold int                 // 超过此字节数走外部存储
}

func (w *ObjectStorageWriter) Write(frame *OutputFrame) error {
    if len(frame.ImageData) > w.threshold {
        key := fmt.Sprintf("%s/%d.jpg", frame.StreamID, frame.SeqNum)
        url, _ := w.storage.Put(key, frame.ImageData)
        frame.ImageData = nil
        frame.Metadata["image_url"] = url
    }
    return w.mq.Write(frame)
}
```

---

## 4. 配置规范

### 4.1 完整配置示例

```yaml
# config.yaml

engine:
  max_concurrent_starts: 5
  shutdown_timeout: 30s
  health_check_interval: 10s
  health_check_timeout: 30s      # 超过此时间未出帧标记 unhealthy

output:
  backend: "kafka"
  kafka:
    brokers:
      - "10.0.1.10:9092"
      - "10.0.1.11:9092"
    topic_prefix: "video-frames"
    max_message_bytes: 1048576   # 1MB
    required_acks: 1             # 0=不等待, 1=leader确认, -1=全部确认
    compression: "snappy"        # none | gzip | snappy | lz4
  serializer:
    format: "jpeg"
    quality: 85
  # Phase 2 预留
  # large_frame:
  #   threshold: 524288
  #   storage_backend: "minio"

streams:
  - id: "gate-north"
    rtsp_url: "rtsp://admin:password@10.0.2.101/stream1"
    capture_fps: 25
    decode_scale: "1920x1080"
    output_topic: "gate-north"
    restart:
      max_retries: 20
      backoff_initial: 1s
      backoff_max: 60s
      backoff_factor: 2.0
    filters:
      - type: "duplicate"
        params:
          threshold: 10
      - type: "noop"

  - id: "lobby-main"
    rtsp_url: "rtsp://admin:password@10.0.2.102/stream1"
    capture_fps: 15
    decode_scale: "1280x720"
    filters:
      - type: "noop"
```

---

## 5. 分阶段实现策略

### 5.1 Phase 1 交付范围

| 模块 | 说明 |
|------|------|
| Stream Manager | 完整实现：配置加载、流生命周期、健康检查、热更新、优雅关闭 |
| Decoder Worker | rawvideo pipe + RawVideoReader |
| Filter Pipeline | 接口骨架 + NoopFilter + DuplicateFilter |
| Output Writer | Kafka（sarama） + JPEG 序列化 |
| Config Manager | YAML 加载 + fsnotify 热更新 |
| Metrics | Prometheus 指标（状态/帧计数/延迟/错误） |
| 部署 | Dockerfile + docker-compose |

### 5.2 Phase 1 明确的预留接口

| 预留点 | 方式 | Phase 2 代价 |
|--------|------|-------------|
| `FrameReader` 接口 | 已定义，Phase 1 只有 RawVideoReader | 新增 JPEGReader ~80 行 |
| `FFmpegCommandBuilder` | args 构建集中在一个函数，预留 hwaccel/pipeFormat 分支 | 加条件分支 ~20 行 |
| `StreamConfig.PipeFormat` | 字段存在于结构体，Phase 1 忽略 | 只改配置解析 |
| `StreamConfig.HWAccel` | 同上 | 同上 |
| `OutputWriter` 接口 | 已定义，Phase 1 只有 KafkaWriter | 新增 ObjectStorageWriter ~100 行 |
| `FrameFilter` 接口 | 已定义，Pipeline 逻辑不变 | 新增 ML Filter ~100 行 |
| `ObjectStorage` 接口 | 可预定义 | Phase 2 实现 |

### 5.3 Phase 2 → Phase 1 切换成本

一个 4K 流从 1080p/rawvideo 切换到 4K/jpeg/GPU加速只需：

1. 配置文件加三行：`pipe_format: "jpeg"`、`hwaccel: "cuda"`、`decode_scale: "3840x2160"`
2. 代码：新增 `JPEGReader` (~80行) + `FFmpegCommandBuilder` 加两个条件分支 (~20行)
3. 测试：新增 JPEG 格式集成测试

总计约 **150 行代码**，不影响 Phase 1 已有逻辑。

---

## 6. 可观测性

### 6.1 Prometheus 指标（Phase 1）

| 指标名 | 类型 | 标签 | 描述 |
|--------|------|------|------|
| `vce_stream_status` | Gauge | stream_id | 1=running, 0=stopped |
| `vce_frames_total` | Counter | stream_id, decision | pass/drop 帧计数 |
| `vce_frame_latency_ms` | Histogram | stream_id | 解码到 Kafka 发送耗时 |
| `vce_decode_errors_total` | Counter | stream_id, error_type | 解码/读取错误 |
| `vce_reconnect_total` | Counter | stream_id | ffmpeg 重连次数 |
| `vce_ffmpeg_restarts_total` | Counter | stream_id | ffmpeg 进程重启次数 |
| `vce_kafka_write_latency_ms` | Histogram | stream_id | Kafka 写入耗时 |
| `vce_kafka_write_errors_total` | Counter | stream_id | Kafka 写入失败次数 |

### 6.2 日志规范

使用结构化日志（`log/slog`），关键事件：

```
level=INFO  msg="stream started"      stream_id=gate-north url=rtsp://...
level=WARN  msg="ffmpeg exited"       stream_id=gate-north exit_code=1 retry=3/20
level=ERROR msg="max retries exceeded" stream_id=gate-north
level=INFO  msg="frame dropped"       stream_id=gate-north reason=backpressure
level=INFO  msg="frame filtered"      stream_id=gate-north filter=duplicate
level=INFO  msg="config reloaded"     added=1 removed=0 modified=2
```

---

## 7. 目录结构

```
VideoStreamCaptureEngine/
├── cmd/
│   └── engine/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── manager/
│   │   ├── manager.go
│   │   └── health.go
│   ├── decoder/
│   │   ├── worker.go
│   │   ├── ffmpeg.go
│   │   ├── reader.go          # FrameReader 接口 + RawVideoReader
│   │   └── reconnect.go
│   ├── filter/
│   │   ├── pipeline.go
│   │   ├── filter.go          # FrameFilter 接口
│   │   ├── noop.go
│   │   └── duplicate.go
│   ├── output/
│   │   ├── writer.go          # OutputWriter 接口
│   │   ├── serializer.go
│   │   └── kafka.go
│   └── metrics/
│       └── metrics.go
├── configs/
│   └── config.example.yaml
├── deploy/
│   ├── Dockerfile
│   └── docker-compose.yaml
├── web/                        # 前端控制台
│   ├── src/
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   ├── components/
│   │   │   ├── Layout.tsx      # 顶部栏 + 侧边栏 + 内容区
│   │   │   ├── StatCard.tsx
│   │   │   ├── StatusDot.tsx
│   │   │   ├── StreamTable.tsx
│   │   │   ├── FilterBar.tsx
│   │   │   └── EventRow.tsx
│   │   ├── pages/
│   │   │   ├── Login.tsx
│   │   │   ├── Dashboard.tsx
│   │   │   ├── StreamList.tsx
│   │   │   ├── StreamDetail.tsx
│   │   │   ├── EngineConfig.tsx
│   │   │   └── EventLog.tsx
│   │   ├── hooks/
│   │   │   └── useMetrics.ts
│   │   └── styles/
│   │       └── theme.css
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── docs/
│   └── superpowers/
│       └── specs/
│           └── 2026-07-10-video-stream-capture-engine-design.md
├── go.mod
├── go.sum
├── Makefile
└── .gitignore
```

---

## 8. Go 依赖选型（Phase 1）

| 用途 | 库 | 理由 |
|------|-----|------|
| Kafka 客户端 | `github.com/IBM/sarama` | Go 生态最成熟的 Kafka 库 |
| YAML 解析 | `gopkg.in/yaml.v3` | 标准选择 |
| 文件监听 | `github.com/fsnotify/fsnotify` | 配置热更新 |
| Prometheus | `github.com/prometheus/client_golang` | 指标暴露 |
| 结构化日志 | `log/slog`（标准库） | Go 1.21+ 内置 |
| 并发管理 | `golang.org/x/sync/errgroup` | goroutine 生命周期管理 |

---

## 9. 风险与对策

| 风险 | 影响 | 对策 |
|------|------|------|
| ffmpeg 进程泄漏（孤儿进程） | 内存/CPU 持续消耗 | Worker 退出时用 process group kill，健康检查兜底 |
| Kafka broker 不可用 | 帧积压丢失 | buffered channel + 丢帧策略；Kafka 重连 |
| 大量流同时重连 | 资源冲击 | `max_concurrent_starts` 限制并发启动数 |
| RTSP 密码明文在配置文件 | 安全风险 | Phase 2 支持环境变量/Secret Manager |
| 单帧 > Kafka max.message.bytes | 消息丢失 | Phase 2 外部存储方案兜底；Phase 1 控制 JPEG 质量 |
| H265 软解打满 CPU | 无法支撑 50 路 | Phase 2 GPU 硬件加速；Phase 1 限定 H264/1080p |

---

## 10. 前端 UI（监控管理控制台）

### 10.1 概述

前端定位为**监控 + 管理控制台**，提供实时流状态监控、流配置管理、事件日志查看的一站式 Web 界面。与后端引擎通过 HTTP API（Prometheus metrics 端点 + 管理 API）通信。

### 10.2 整体导航结构

```
┌──────────────────────────────────────────────────────┐
│  顶部全局导航栏                                       │
│  Logo │ 系统状态 │ 运行时长 │ 在线数 │ 🔍搜索 │ 🔔 │ ⚙️ │ 👤 │
├────────┬─────────────────────────────────────────────┤
│        │                                              │
│  📊 仪表盘  │        内容区                             │
│  📹 流管理  │        (页面内容)                         │
│  ⚙️ 引擎配置│                                         │
│  📋 事件日志│                                         │
│  👤 用户    │                                         │
│        │                                              │
└────────┴─────────────────────────────────────────────┘
```

- **顶部导航栏**：贯穿全宽，显示系统状态指示灯、运行时长、在线统计、全局搜索、通知铃铛（未读告警数）、设置入口、用户头像
- **左侧边栏**：5 个页面入口（仪表盘 / 流管理 / 引擎配置 / 事件日志 / 用户），当前页高亮
- **内容区**：页面主体，右侧滚动

### 10.3 仪表盘（Dashboard）

**目标**：运维人员打开即见全局状态，快速发现异常。

**组成**：

| 区域 | 内容 | 说明 |
|------|------|------|
| 顶部统计卡片 | 在线数/总数、今日出帧、平均 FPS、活跃告警 | 4 个等宽卡片，颜色区分（绿/蓝/黄/红） |
| FPS 趋势图 | 全系统 FPS 聚合曲线 | 时间范围切换（5m/15m/1h），断流异常标记 |
| 最近事件 | 最新 3 条未确认/已确认事件 | 按级别着色（红/黄/灰），可点击查看详情 |

### 10.4 流管理（Stream List）

**目标**：集中管理所有 RTSP 流的配置和运行状态。

**功能**：
- **搜索与过滤**：按流 ID / RTSP 地址搜索，按状态（全部/运行中/告警/已停止）、分组、分辨率、排序过滤
- **流列表表格**：流 ID、分组标签、状态指示灯（绿/黄/红/灰）、FPS、分辨率、累计出帧数、延迟、操作按钮
- **操作**：单流启停/重启，批量选中后启动/停止/重启
- **批量导入**：通过弹窗上传 YAML/JSON/CSV 文件或粘贴配置，支持模板下载，导入结果汇总（成功/跳过/失败详情）
- **添加流**：单流配置表单
- **导出**：当前筛选结果导出为 YAML/CSV

### 10.5 单流详情（Stream Detail）

**目标**：深入查看单路流的所有运行数据和配置。

**组成**：

| 区域 | 内容 |
|------|------|
| 面包屑 | ← 流列表 / stream-id，状态指示灯 + 重启/停止/保存按钮 |
| 帧预览 | 实时帧缩略图（~2s 刷新），叠加时间戳和帧序号，支持暂停/放大 |
| 实时指标 | FPS、延迟（P99）、累计出帧、丢帧率，四宫格卡片 |
| 配置摘要 | RTSP URL、分辨率、采集帧率、输出 Topic、传输格式、分组，可编辑 |
| FPS 趋势图 | 单流 FPS 折线图，时间范围切换（15m/1h/6h/24h） |
| 过滤链管线 | 输入 → Filter1 → Filter2 → Kafka 输出，各级吞吐量和丢弃率可视化 |
| 本流事件 | 该流的时间线事件列表（配置变更、断流/恢复、丢帧） |

### 10.6 引擎配置（Engine Config）

**目标**：管理全局引擎参数，修改后保存并重启生效（部分支持热更新）。

**配置区**：
- **引擎参数**：最大并发启动数、关闭超时、健康检查间隔/超时
- **Kafka 输出**：Broker 地址（多地址逗号分隔）、Topic 前缀、最大消息大小、ACK 级别、压缩方式
- **序列化**：输出格式（jpeg/png/webp）、JPEG 质量
- **默认重启策略**：最大重试次数、初始退避、最大退避、倍增因子 — 附带指数退避曲线可视化
- **YAML 预览**：右侧实时显示等效 YAML 配置
- **操作**：保存配置、恢复默认、导出 YAML

### 10.7 事件日志（Event Log）

**目标**：全系统审计日志，支持快速定位问题。

**功能**：
- **过滤**：按级别（全部/错误/警告/信息/恢复）、按流名、按时间范围、按关键词搜索
- **表格**：时间、级别标签（彩色 Tag）、流名、事件描述、确认状态（未确认高亮）、详情入口
- **操作**：全部确认、导出日志

### 10.8 用户管理（Login）

**目标**：简单登录认证，保护控制台不被未授权访问。

**功能**：
- **登录页**：用户名 + 密码表单，登录失败提示，无注册功能（账号由配置文件或环境变量预设）
- **会话管理**：JWT token，存储在 HttpOnly Cookie，支持过期自动跳转登录页
- **顶部栏集成**：登录后右上角显示当前用户名，点击可退出登录
- **路由守卫**：未登录访问任何页面均重定向到登录页

**登录页布局**：
- 居中卡片式布局，深色背景
- Logo + 产品名称 "CaptureEngine"
- 用户名 / 密码输入框 + "登录" 按钮
- 错误提示（用户名或密码错误）

**后端认证**：
- 引擎启动时从配置文件或环境变量读取预设账号（Phase 1 单账号）
- 提供 `/api/login` 端点，验证后签发 JWT
- 所有管理 API 通过中间件验证 JWT
- Phase 2 可扩展为多用户 + 角色权限

### 10.9 前端技术栈（Phase 1 建议）

| 用途 | 选型 | 理由 |
|------|------|------|
| 框架 | React + TypeScript | 生态成熟，状态管理灵活 |
| 构建 | Vite | 开发体验好，HMR 快 |
| UI 组件库 | 自定义（基于 CSS Variables 暗色主题） | 轻量无依赖，风格统一 |
| 图表 | Recharts 或 uPlot | FPS 趋势图 |
| 数据获取 | SWR 或 React Query | 轮询 Prometheus API |
| 状态 | React Context + useReducer | 页面级状态即可 |
| 部署 | 静态文件内嵌到 Go binary（embed.FS） | 单二进制部署 |

### 10.10 目录结构（前端）

```
web/
├── src/
│   ├── App.tsx
│   ├── main.tsx
│   ├── components/
│   │   ├── Layout.tsx          # 顶部栏 + 侧边栏 + 内容区
│   │   ├── StatCard.tsx        # 统计卡片
│   │   ├── StatusDot.tsx       # 状态指示灯
│   │   ├── StreamTable.tsx     # 流列表表格
│   │   ├── FilterBar.tsx       # 搜索过滤栏
│   │   └── EventRow.tsx        # 事件行
│   ├── pages/
│   │   ├── Login.tsx           # 登录页
│   │   ├── Dashboard.tsx       # 仪表盘
│   │   ├── StreamList.tsx      # 流管理
│   │   ├── StreamDetail.tsx    # 单流详情
│   │   ├── EngineConfig.tsx    # 引擎配置
│   │   └── EventLog.tsx        # 事件日志
│   ├── hooks/
│   │   └── useMetrics.ts       # 轮询 Prometheus API
│   └── styles/
│       └── theme.css           # CSS Variables 暗色主题
├── index.html
├── package.json
├── tsconfig.json
└── vite.config.ts
```
