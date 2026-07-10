# Lightweight Frame Filtering & Tracking — 技术设计方案

**版本**: v1.0  
**日期**: 2026-07-10  
**状态**: Draft  
**依赖**: VideoStreamCaptureEngine Phase 1（已完成）

---

## 1. 概述

在现有引擎的 `FilterPipeline` 上扩展两层能力：

| 阶段 | 功能 | 实现 |
|------|------|------|
| Stage 1: 质量过滤 | 排除模糊帧 + 排除重复帧 | `BlurFilter`（拉普拉斯方差）+ `DuplicateFilter`（dHash，已实现） |
| Stage 2: 轻量跟踪 | 人脸检测+跟踪（优先），预留物体跟踪接口 | `FaceTracker`（Pigo 纯 Go）+ `TrackerFilter` 包装器 |
| Stage 3: Kafka 输出 | 带检测元数据的帧发布到 Kafka | 扩展现有 `KafkaWriter` 的 metadata 字段 |
| Stage 4: 精确识别（下游） | YOLO/RetinaFace 等重模型 | **不在引擎范围内**，独立 Python 服务消费 Kafka |

---

## 2. 整体管线

```
RTSP → ffmpeg → FrameReader
                    │
                    ▼
┌─────────────────────────────────────────────┐
│  Stage 1: 质量过滤                           │
│                                              │
│  BlurFilter (3ms) → DuplicateFilter (1ms)    │
│  丢弃模糊               丢弃重复               │
│                                              │
│  输出: ~22fps 清晰帧                          │
└────────────────────┬────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────┐
│  Stage 2: 轻量跟踪（仅每 N 帧触发）             │
│                                              │
│  MotionGate: 帧差 → 无变化? 复用上次Detection  │
│  有变化? → resize 640×360                     │
│         → ROI 裁剪(如有上一帧位置)              │
│         → Pigo 检测 (~5ms)                    │
│         → IOU 匹配 → track_id                 │
│                                              │
│  输出: 每帧都带 Detection（检测帧实际检测，      │
│         中间帧复用上次结果）                     │
└────────────────────┬────────────────────────┘
                     │
                     ▼
                 KafkaWriter
                     │
                     ▼
┌─────────────────────────────────────────────┐
│  Stage 3: 下游精确识别                        │
│  Python: YOLOv8 / RetinaFace / DeepSORT     │
└─────────────────────────────────────────────┘
```

### 性能预算（单路 1080p@25fps）

| 操作 | 频率 | 耗时 |
|------|------|------|
| Blur (Laplacian) | 每帧 | 3ms |
| Duplicate (dHash) | 每帧 | 1ms |
| MotionGate (帧差) | 每帧 | 0.5ms |
| Pigo (resize+detect) | 每4帧 | 5ms × 0.25 |
| IOU 匹配 | 每4帧 | 0.1ms |
| **总计** | — | **~6ms/帧** |

50 路总 CPU：6ms × 25fps × 50 = 7.5 核。可行。

---

## 3. 组件设计

### 3.1 BlurFilter — 拉普拉斯方差模糊检测

**算法**: 3×3 Laplacian 卷积核 → 方差计算

```
kernel = [0,  1, 0]
         [1, -4, 1]
         [0,  1, 0]
```

- 先转为灰度：`Gray = 0.299R + 0.587G + 0.114B`
- 3×3 卷积 → Laplacian 响应图
- 计算响应图方差
- 方差 < threshold → 模糊 → 丢弃

```go
type BlurFilter struct {
    threshold float64
}

func (b *BlurFilter) Name() string { return "blur" }

func (b *BlurFilter) Apply(frame image.Image, meta FrameMeta) (FilteredFrame, FilterDecision) {
    score := laplacianVariance(frame)
    if score < b.threshold {
        return FilteredFrame{}, FilterDrop
    }
    return FilteredFrame{Image: frame, Meta: meta, Score: score}, FilterPass
}
```

**配置**: `threshold: 100`（默认，值越低越严格）

### 3.2 Tracker 接口 & Detection

```go
type Detection struct {
    Class      string          // "face" | "person" | "vehicle" | ...
    BBox       image.Rectangle
    Confidence float64
    TrackID    int
}

type Tracker interface {
    Track(frame image.Image) []Detection
    Close() error
}
```

**设计决策**: `Tracker` 是独立接口（非 `FrameFilter`），由 `TrackerFilter` 包装为 `FrameFilter`。这样后续扩展 ObjectTracker 时只需实现 `Tracker` 接口，不影响管线逻辑。

### 3.3 TrackerFilter — 包装器

```go
type TrackerFilter struct {
    tracker      Tracker
    minDetections int
}

func (t *TrackerFilter) Apply(frame image.Image, meta FrameMeta) (FilteredFrame, FilterDecision) {
    detections := t.tracker.Track(frame)
    if len(detections) < t.minDetections {
        return FilteredFrame{}, FilterDrop
    }
    return FilteredFrame{
        Image: frame,
        Meta:  meta,
        Score: 1.0,
    }, FilterPass
}
```

### 3.4 FaceTracker — Pigo 纯 Go 实现

**库**: `github.com/esimov/pigo` — 像素强度比较级联分类器

**优化策略**:

1. **降采样**: 原帧 resize 到 640×360 再检测
2. **帧抽检**: 每 4 帧检测一次，中间帧复用上次 Detection（16fps 抽检 → 4fps 检测）
3. **ROI 复用**: 有上一帧位置时，仅在扩张 1.5x 区域内搜索
4. **MotionGate**: 帧差法快速判断是否有变化，完全静止则跳过检测

```go
type FaceTracker struct {
    classifier   *pigo.Pigo
    lastDetections []Detection
    lastFrame     *image.Gray
    frameCount    int
    detectEvery   int       // 默认 4
    trackMaxLost  int       // 丢失多少帧分配新 track_id
    iouThreshold  float64
    nextTrackID   int
    tracks        map[int]*trackState
    cfg           FaceTrackerConfig
}

type FaceTrackerConfig struct {
    CascadeFile   string
    MinConfidence float64
    MinSize       int
    MaxSize       int
    DetectEvery   int
    TrackMaxLost  int
    IOUThreshold  float64
}
```

### 3.5 IOU 匹配 + Track 管理

```go
type trackState struct {
    id         int
    lastBBox   image.Rectangle
    lostFrames int
}

func (ft *FaceTracker) matchDetections(detections []Detection) {
    // 1. 计算 detections 与现有 tracks 的 IOU 矩阵
    // 2. 贪心匹配: IOU > threshold 的分配已有 track_id
    // 3. 未匹配的 detection → 新 track_id
    // 4. 未匹配的 track → lostFrames++
    // 5. lostFrames > maxLost → 清理 track
}
```

### 3.6 ObjectTracker — 预留接口

```go
type ObjectTracker struct{}

func (o *ObjectTracker) Track(frame image.Image) []Detection {
    return nil  // Phase 2 实现
}

func (o *ObjectTracker) Close() error { return nil }
```

---

## 4. Kafka 消息格式

```json
{
  "stream_id": "gate-north",
  "seq_num": 142,
  "timestamp": "2026-07-10T14:32:05Z",
  "image_data": "<jpeg bytes>",
  "image_size": 284712,
  "quality": 0.95,
  "metadata": {
    "blur_score": "245.3",
    "detections": "[{\"class\":\"face\",\"bbox\":[120,80,200,200],\"conf\":0.92,\"track_id\":5}]",
    "detection_count": "1"
  }
}
```

---

## 5. 配置

```yaml
streams:
  - id: "gate-north"
    rtsp_url: "rtsp://..."
    filters:
      - type: "blur"
        params:
          threshold: 100
      - type: "duplicate"
        params:
          threshold: 10
      - type: "tracker"
        params:
          tracker_type: "face"
    tracker:
      type: "face"
      params:
        cascade_file: ""                    # 留空使用 Pigo 内置
        min_confidence: 0.5
        min_size: 80
        max_size: 1000
        detect_every: 4                     # 每 N 帧检测一次
        track_max_lost: 5
        iou_threshold: 0.3
```

---

## 6. 新增文件

```
internal/
├── filter/
│   ├── blur.go              # BlurFilter（~100 行）
│   └── tracker_filter.go    # TrackerFilter 包装器（~50 行）
├── tracker/
│   ├── tracker.go           # Tracker 接口 + Detection 结构
│   ├── face.go              # Pigo FaceTracker（~200 行）
│   ├── iou.go               # IOU 匹配（~60 行）
│   └── object.go            # ObjectTracker 预留实现（~15 行）
```

---

## 7. 依赖

| 用途 | 库 | 理由 |
|------|-----|------|
| 人脸检测 | `github.com/esimov/pigo` | 纯 Go，零 CGO，级联分类器，轻量快速 |

**不引入 OpenCV**：Pigo 是纯 Go 实现，无外部依赖，编译部署简单。

---

## 8. 风险与对策

| 风险 | 对策 |
|------|------|
| Pigo 精度不如 MTCNN/RetinaFace | 设计意图即轻量预过滤，Stage 3 下游服务做精确检测 |
| 帧抽检导致目标丢失 | `trackMaxLost=5` 允许短暂丢失，IOU 匹配合并断续检测 |
| 暗光场景人脸漏检 | 阈值可配置，下游 Stage 3 全帧检测兜底 |
| gocv 依赖引入编译复杂度 | 不使用 gocv，全部纯 Go |
