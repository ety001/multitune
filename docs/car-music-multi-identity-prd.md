# 多音盒 MultiTune — 车机多身份音乐播放器 PRD

> 中文名：多音盒  
> 英文名：MultiTune  
> 应用方向：懒猫微服 LPK 轻应用，主要在车机浏览器/WebView 中运行。  
> 核心痛点：车内多人用车时，现有音乐应用无法快速切换“身份 + 歌单”。  
> 文档版本：v0.3（技术方案与需求细化）  
> 更新时间：2026-07-14

---

## 一、项目概述

### 1.1 背景
懒猫微服当前音乐类应用在车机场景下存在明显体验缺口：
- 家庭成员/不同驾驶者共用一辆车时，歌单、喜好、播放进度混在一起。
- 切换账号或歌单步骤过多，行驶中操作不安全。
- 车机屏幕以触控为主，需要大按钮、少层级、夜间友好的界面。

### 1.2 产品目标
做一款**以“身份快速切换”为核心**的车载音乐播放器：
1. 上车后 1 步切换身份（大按钮/手势）。
2. 每个身份拥有独立歌单列表与播放记忆。
3. 播放器功能极简但完整：播放/暂停、上下曲、列表、顺序/随机/单曲循环、进度记忆。
4. 优先适配懒猫微服 LPK 运行环境与车机屏幕。

### 1.3 目标用户
- 家庭共享一辆车的多驾驶者。
- 追求“上车即听、换人即切”的车机用户。
- 不习惯复杂音乐 App 的驾驶者。

---

## 二、应用命名

**已确定**：

- 中文名：**多音盒**
- 英文名：**MultiTune**
- 懒猫包名：`ink.akawa.ety001.multitune`

**命名含义**：
- “多”对应 Multi，强调多身份。
- “音盒”寓意每个身份都是一个独立的音乐盒子，呼应 MultiTune 的音乐属性。
- MultiTune 直接表达“多身份、多曲调”。

> 此前候选：多面音乐、车乐切、身份唱片、一键音驾、音匣、多身份播放器、座舱曲库、速切音乐。最终选定“多音盒 / MultiTune”。

---

## 三、功能需求

### 3.1 身份管理（核心）

| 功能 | 说明 | 优先级 |
|---|---|---|
| 创建身份 | 自定义身份名称（如“爸爸”、“妈妈”、“通勤”、“跑山”），选择身份颜色 | P0 |
| 身份列表 | 首页横向或网格展示所有身份，大按钮，一触切换 | P0 |
| 快速切换 | 任何页面都能快速回到身份列表或呼出身份切换浮层 | P0 |
| 编辑身份 | 改名、换颜色、调整顺序 | P1 |
| 删除身份 | 删除身份及其下的歌单与播放记忆（可提示确认） | P1 |
| 默认身份 | 可设置某个身份为上车默认身份 | P1 |

**默认身份逻辑**：
- 创建第一个身份时自动设为默认身份。
- 手动设置某身份为默认时，自动取消其他身份的默认状态。
- 删除默认身份时，自动将排序最前的身份设为默认（若还有身份）。

**核心交互原则**：
- 身份切换 ≤ 2 次点击。
- 车机模式下身份卡片尺寸 ≥ 120×120 px，触控容错高。

### 3.2 歌单管理

| 功能 | 说明 | 优先级 |
|---|---|---|
| 创建歌单 | 每个身份下可创建多个歌单，命名自定义 | P0 |
| 文件浏览器（完整版） | PC/大屏端使用：树形目录、多选、拖拽、批量添加、搜索 | P0 |
| 文件浏览器（精简版） | 车机/移动端使用：层级导航、大按钮、单选/简单多选、已选预览 | P1 |
| 添加歌曲 | 从挂载的多个存储源中选择文件加入歌单，歌单只保存文件路径索引 | P0 |
| 编辑歌单 | 改名、调整歌曲顺序、删除歌曲引用 | P1 |
| 删除歌单 | 删除歌单及内部歌曲索引（不删除源文件） | P1 |
| 最近播放 | 自动生成最近播放歌单/记录 | P2 |

**歌单本质**：歌单是歌曲文件路径的索引集合，不复制音乐文件。同一首歌可被多个歌单引用。

### 3.3 播放器（弱化但完整）

| 功能 | 说明 | 优先级 |
|---|---|---|
| 播放/暂停 | 大按钮，支持空格/方向盘按键预留 | P0 |
| 上一曲 / 下一曲 | 大按钮，点击反馈明显 | P0 |
| 播放列表 | 当前歌单歌曲列表，点击切歌 | P0 |
| 播放模式 | 顺序播放、随机播放、单曲循环 | P0 |
| 进度条 | 可拖动，显示当前时间/总时长 | P0 |
| 音量控制 | 媒体音量调节滑块（`audio.volume`），按身份记忆偏好 | P0 |
| 封面显示 | 读取歌曲内嵌封面或歌单默认封面 | P2 |

### 3.4 记忆功能

| 功能 | 说明 | 优先级 |
|---|---|---|
| 歌单播放记忆 | 记住每个身份最后播放的歌单与歌曲 | P0 |
| 歌曲进度记忆 | 记住每首歌上次播放到的位置，再次播放时询问/自动续播 | P0 |
| 播放模式记忆 | 记住每个身份的播放模式偏好 | P1 |
| 界面状态记忆 | 夜间/日间模式、身份列表排序等 | P2 |

---

## 四、用户流程

### 4.1 首次使用
1. 打开应用 → 进入“创建第一个身份”引导页。
2. 输入身份名称、选择身份颜色。
3. 创建第一个歌单，通过文件浏览器从已挂载的存储源中选择歌曲加入。
4. 进入播放器，开始播放。

### 4.2 日常使用（车机场景）
1. 上车打开应用 → 首页展示身份卡片。
2. 点击身份 → 进入该身份默认歌单/最后播放位置，自动续播。
3. 点击“切换身份”按钮 → 回到身份列表 → 点击新身份 → 秒切。
4. 播放器页可进行播放控制、切歌、切换播放模式。

### 4.3 管理场景（PC/大屏）
1. 在 PC 浏览器打开应用 → 进入完整版管理界面。
2. 使用完整版文件浏览器浏览多存储源、批量选择歌曲、创建/编辑歌单。
3. 车机端仅保留精简版浏览与播放，避免驾驶中复杂操作。

### 4.4 后台/锁屏场景
- 车机 WebView 切后台时，尽量保持播放状态（依赖浏览器 Audio 策略）。
- 提供“驾驶模式”全屏简化界面，只保留播放控制与身份切换入口。

---

## 五、界面与交互

### 5.1 页面结构

**车机/移动端（精简版）**：
```
首页（身份选择）
  └── 身份卡片网格/横向列表
      └── 点击身份 → 歌单页
          └── 点击歌单 → 播放器页
              └── 播放控制 + 歌曲列表 + 模式切换
```

**PC/大屏（完整版）**：
```
管理首页
  ├── 身份管理
  ├── 歌单管理
  │     └── 文件浏览器 → 批量选歌 → 创建/编辑歌单
  └── 播放器
```

### 5.2 设计原则
- **车机以播放和身份切换为核心**：管理功能（创建歌单、批量选歌）主要在 PC/大屏端完成。
- **大**：按钮、卡片、文字足够大，适合车机触控与远距离观看。
- **少**：每屏不超过 3 个主要操作。
- **暗**：默认深色主题，减少夜间驾驶眩光。
- **快**：切换身份、切歌都有明确动画反馈，但不过度。

### 5.3 关键界面描述

#### 身份选择页
- 顶部：应用 Logo + 设置入口 + 夜间/日间切换。
- 中部：身份卡片网格（2×2 或 1×N 横向滑动）。
- 每个卡片：颜色块 + 身份名称 + 最后播放歌单名称。
- 底部：“+ 新建身份”大按钮。

#### 歌单页（车机精简版）
- 顶部：当前身份颜色块/名称 + “切换身份”大按钮。
- 中部：该身份下的歌单列表（大卡片）。
- 底部：最近播放歌单。
- **不强调新建/编辑歌单**，这些操作引导到 PC 完整版。

#### 播放器页
- 顶部：返回歌单页 + 当前歌单名称 + “切换身份”入口。
- 中部：歌曲封面/默认封面 + 歌曲名 + 歌手。
- 中下部：进度条 + 时间。
- 底部：上一曲、播放/暂停、下一曲大按钮。
- 最底部：播放模式切换 + 播放列表展开按钮。

#### 文件浏览器（完整版，PC/大屏）
- 左侧：存储源树（主目录 / USB / SMB 共享）。
- 中部：文件列表，支持多选、全选、按文件夹批量选择。
- 右侧：已选歌曲预览 + 添加到歌单。
- 顶部：路径面包屑、搜索框。
- 操作：创建歌单、追加到已有歌单、取消。

#### 文件浏览器（精简版，车机/移动）
- 顶部：当前路径 + 返回上一级大按钮。
- 中部：文件夹/文件列表，大按钮。
- 底部：已选数量 + “添加到当前歌单”大按钮。
- 限制：不支持复杂多选，优先单文件或整文件夹添加。

#### 驾驶模式（可选 P1）
- 全屏简化播放器。
- 超大播放/暂停、上一曲/下一曲。
- 屏幕边缘滑动切换身份（左滑/右滑）。

---

## 六、技术架构

### 6.1 部署形态
- 懒猫微服 LPK 轻应用。
- **后端**：Go + Gin 提供 REST API、音频流服务、SQLite 数据持久化。
- **前端**：双版本设计。
  - **现代版**：Vue 3 + Vite，面向现代浏览器（Chrome/WebView ≥ 75）。
  - **简化版**：纯 HTML + ES5 JS + 原生 DOM，专为比亚迪等老版本车机浏览器（Chrome/WebView ≤ 74）兼容。
- 入口页自动检测浏览器版本并跳转对应版本。
- 音乐源文件：不固定单一目录，容器内挂载多个存储源（懒猫主目录、USB 存储、SMB 共享等），后端统一抽象为虚拟文件树。

### 6.2 后端技术栈

| 项目 | 选型 | 说明 |
|---|---|---|
| 语言 | Go 1.23+ | 静态编译、部署简单、资源占用低 |
| Web 框架 | Gin | 轻量、路由清晰、中间件丰富 |
| 数据库 | SQLite (modernc.org/sqlite) | 零配置、单文件、CGO-free 静态编译友好 |
| 数据库迁移 | golang-migrate / 手写 migrate | 版本化管理表结构 |
| 音频元数据 | taglib / ffprobe | 读取 ID3/封面/时长 |
| 音频流 | `http.ServeContent` / `io.Copy` | 支持 Range 请求、进度拖动 |
| 日志 | slog / zap | 结构化日志 |
| 配置 | Viper / envconfig | 环境变量 + 默认值 |

**为什么不直接用 Next.js API Routes**：
- 项目已决定用 Go 后端，职责更清晰（API + 音频流 + 文件扫描）。
- Go 二进制静态编译后更适合 LPK 容器，镜像更小、启动更快。

### 6.3 前端技术栈

#### 现代版
- **框架**：Vue 3（Composition API）
- **构建**：Vite
- **状态管理**：Pinia
- **路由**：Vue Router
- **HTTP 客户端**：axios / fetch
- **音频播放**：HTML5 `<audio>`（原生事件足够，无需 Howler.js 额外依赖）
- **UI 组件**：手写组件，车机优先

#### 简化版（老车机兼容）
- **技术**：纯 HTML + 内嵌 `<style>` + 内联 ES5 JS
- **DOM 操作**：原生 `document.getElementById` + `innerHTML` 拼接
- **网络请求**：原生 `XMLHttpRequest`
- **兼容性**：
  - 不使用 ES6+ 语法（箭头函数、const/let、Promise、async/await 等全部避免）。
  - 不使用 CSS Flexbox/Grid 复杂布局，使用 `display: block` + 固定/百分比宽度。
  - 避免 `addEventListener` 的 `{ once: true }` 等高级选项。
  - 不依赖外部框架，全部内联在 HTML 中。

> 参考 `~/workspace/lzc-story/src/app/simple/` 的实现方式： landing 页检测 Chrome/WebView 版本，≤ 74 自动跳转到 `/simple/` 路径下的简化版页面。

#### 浏览器版本检测策略
```javascript
function isOldBrowser() {
  var ua = navigator.userAgent;
  var chromeMatch = ua.match(/Chrome\/(\d+)/);
  var webViewMatch = ua.match(/wv.*Chrome\/(\d+)/);
  var chromeVersion = chromeMatch ? parseInt(chromeMatch[1], 10) : null;
  var webViewVersion = webViewMatch ? parseInt(webViewMatch[1], 10) : null;
  return (webViewVersion && webViewVersion <= 74) || (chromeVersion && chromeVersion <= 74);
}
```

### 6.4 数据模型

```yaml
Identity（身份）:
  - id: string            # UUID
  - name: string          # 身份名称
  - avatar_color: string  # 身份颜色（卡片背景色）
  - sort_order: int       # 排序
  - is_default: bool      # 是否默认身份
  - created_at: int       # 时间戳
  - updated_at: int

Playlist（歌单）:
  - id: string
  - identity_id: string   # 所属身份
  - name: string
  - cover_url?: string
  - sort_order: int
  - created_at: int
  - updated_at: int

Song（歌曲，引用）:
  - id: string
  - path: string          # 容器内绝对路径，如 /music/xxx.mp3
  - source: string        # 来源标识：home / usb / smb 等
  - title: string
  - artist: string
  - album: string
  - duration: int         # 秒
  - cover_url?: string    # 封面 URL

PlaylistSong（歌单-歌曲关联）:
  - playlist_id: string
  - song_id: string
  - sort_order: int

PlaybackState（播放状态，每个身份一条）:
  - identity_id: string
  - playlist_id: string
  - song_id: string
  - position: int         # 秒
  - mode: string          # 'order' | 'random' | 'single-loop'
  - updated_at: int

SongProgress（单曲进度记忆，满足 P0 需求）:
  - identity_id: string   # 联合主键
  - song_id: string       # 联合主键
  - position: int         # 该歌曲上次播放到的位置（秒）
  - updated_at: int
```

**数据模型说明**：
- `playback_states` 记录每个身份**当前正在播放**的位置，用于上车后自动续播。
- `song_progress` 记录每个身份下**每首歌**上次播放到的位置，用于切换回某首歌时从该位置续播。
- 播放过程中同时更新两张表：当前播放进度写入 `playback_states`，当前歌曲进度也写入 `song_progress`。

### 6.5 后端 API 设计（初稿）

#### 身份 API
```
GET    /api/identities              # 获取身份列表
POST   /api/identities              # 创建身份
GET    /api/identities/:id          # 获取身份详情
PUT    /api/identities/:id          # 更新身份
DELETE /api/identities/:id          # 删除身份
POST   /api/identities/:id/default  # 设为默认身份
```

#### 歌单 API
```
GET    /api/identities/:id/playlists     # 获取身份下的歌单
POST   /api/identities/:id/playlists     # 创建歌单（归属于指定身份）
GET    /api/playlists/:id               # 获取歌单详情（含歌曲）
PUT    /api/playlists/:id               # 更新歌单
DELETE /api/playlists/:id               # 删除歌单
POST   /api/playlists/:id/songs         # 添加歌曲到歌单
DELETE /api/playlists/:id/songs/:songId # 从歌单移除歌曲
PUT    /api/playlists/:id/songs/order   # 调整歌单内歌曲顺序
```

#### 歌曲与扫描 API
```
POST   /api/scan                # 扫描指定目录，返回发现的歌曲
GET    /api/songs               # 歌曲列表/搜索
GET    /api/songs/:id           # 歌曲详情
GET    /api/songs/:id/cover     # 歌曲封面
GET    /api/songs/:id/stream   # 音频流（支持 Range）
```

#### 文件浏览器 API
```
GET    /api/fs/sources          # 获取已挂载的存储源列表
GET    /api/fs/list?path=xxx    # 列出指定路径下的文件/文件夹
GET    /api/fs/search?q=xxx     # 跨存储源搜索歌曲
```

存储源示例响应：
```json
[
  { "id": "root", "name": "根目录", "path": "/", "available": true }
]
```

#### 播放状态 API
```
GET    /api/playback/:identityId       # 获取身份播放状态
POST   /api/playback/:identityId       # 保存/更新播放状态
```

#### 通用
```
GET    /api/healthz            # 健康检查
POST   /api/device-info        # 接收并记录车机设备信息（参考 lzc-story）
```

### 6.6 音频流方案

老版本 WebView 对直接播放 `file://` 路径支持不稳定，必须由后端提供音频流：

```
GET /api/songs/:id/stream
```

实现要点：
- 读取 `/lzcapp/...` 下的真实文件。
- 支持 HTTP Range 请求，允许拖动进度条。
- 根据文件扩展名设置正确的 `Content-Type`：
  - `audio/mpeg` (mp3)
  - `audio/flac` (flac)
  - `audio/mp4` (m4a)
  - `audio/aac` (aac)
  - `audio/ogg` (ogg)
  - `audio/wav` (wav)
- 现代浏览器可同样使用该接口，统一播放体验。

---

## 七、懒猫微服 LPK 适配要点

### 7.1 文件挂载
多音盒不再锁定统一的媒体根目录，在 `lzc-manifest.yml` 中按实际需求声明任意 `binds`，后端把每个挂载点都当作可浏览路径：

```yaml
services:
  multitune:
    image: registry.lazycat.cloud/xxx/xxx/multitune:<hash>
    binds:
      # 懒猫微服用户主目录
      - /lzcapp/home:/music/home:ro
      # USB 外接存储
      - /lzcapp/var/usb:/music/usb:ro
      # SMB 共享挂载点
      - /lzcapp/var/smb:/music/smb:ro
      # 应用数据（数据库、配置、播放状态）
      - /lzcapp/var/multitune/data:/app/data
```

用户添加歌曲时，歌单只保存容器内绝对路径索引，不复制文件。前端从根目录进入后，选择上述挂载目录即可扫描歌曲。

### 7.2 网络与访问
- 通过 `application.routes` 暴露 Web UI：
  ```yaml
  application:
    subdomain: multitune
    routes:
      - /=http://multitune:8080
  ```

### 7.3 车机适配
- 考虑横屏优先（车机多为横屏）。
- 触摸目标最小 64×64 px，推荐 96×96 px 以上。
- 禁用页面缩放（viewport 固定）。
- 支持键盘事件（空格播放/暂停、方向键切歌）。

### 7.4 权限
- `net.internet`：当前版本不需要，所有功能离线可用（歌词/封面下载暂不考虑）。
- `user.notify`：播放状态通知（可选，V1.1 后考虑）。
- 文件读取权限通过 `binds` 挂载实现。

### 7.5 Dockerfile 多阶段构建（示例）

```dockerfile
# Stage 1: 构建 Vue 3 现代版前端
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
RUN corepack enable && corepack prepare pnpm@latest --activate
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

# Stage 2: 构建 Go 后端（CGO-free，使用 modernc.org/sqlite）
FROM golang:1.23-alpine AS backend-builder
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w" \
    -o /multitune-server ./cmd/server

# Stage 3: 运行时（alpine 精简镜像）
FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl ffmpeg

COPY --from=backend-builder /multitune-server /usr/local/bin/multitune-server
COPY --from=frontend-builder /app/frontend/dist /app/static
COPY simple/ /app/static/simple/

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/healthz || exit 1
ENTRYPOINT ["/usr/local/bin/multitune-server"]
```

说明：
- 使用 `modernc.org/sqlite`（CGO-free），`CGO_ENABLED=0` 静态编译，无需 gcc。
- 前端用 pnpm 替代 npm，符合项目统一包管理器。
- 运行时改用 alpine（约 5MB），配合 ffmpeg apk 包，总镜像体积显著小于 debian 方案。
- `simple/` 目录下的纯 HTML 文件直接复制到 `/app/static/simple/`，由 Go 静态文件中间件服务。
- Vue 构建产物挂载到 `/app/static/`，入口 `index.html` 内嵌浏览器检测脚本。

### 7.6 lzc-manifest.yml 示例

```yaml
application:
  subdomain: multitune
  run_as: "1000:1000"
  routes:
    - /=http://multitune:8080

services:
  multitune:
    image: registry.lazycat.cloud/xxx/xxx/multitune:<hash>
    binds:
      - /lzcapp/home:/music/home:ro
      - /lzcapp/var/usb:/music/usb:ro
      - /lzcapp/var/smb:/music/smb:ro
      - /lzcapp/var/multitune/data:/app/data
    environment:
      - DATA_PATH=/app/data
      - PORT=8080
```

### 7.7 构建与安装命令

```bash
# 1. 构建镜像并推送到 Docker Hub
docker buildx build --builder ssd-builder \
  --platform linux/amd64 \
  -t ety001/multitune:latest-$(git rev-parse --short HEAD) \
  --push .

# 2. copy-image 到懒猫 registry
lzc-cli appstore copy-image ety001/multitune:latest-$(git rev-parse --short HEAD)

# 3. 更新 lzc-manifest.yml 中的 image 字段

# 4. 构建并安装 LPK
cd ~/workspace/lzc-appdb/multitune
lzc-cli project build
lzc-cli app install
```

---

## 八、非功能需求

| 维度 | 要求 |
|---|---|
| 性能 | 身份切换 < 300ms；歌单加载 < 1s（1000 首以内）。 |
| 离线可用 | 除首次扫描外，播放与切换不依赖网络。 |
| 数据安全 | SQLite 数据库文件定期备份到 `/app/data/backup/`（对应挂载点 `/lzcapp/var/multitune/data/backup/`）。 |
| 可维护性 | 代码结构清晰，前端/后端分离，便于后续扩展。 |
| 可访问性 | 高对比度、大字体、支持键盘/方向盘按键扩展。 |

---

## 九、版本规划

### MVP（第一阶段）
- 身份 CRUD + 首页大卡片切换。
- 歌单 CRUD + 从挂载目录添加歌曲。
- 基础播放器：播放/暂停、上下曲、进度条、三种播放模式。
- 歌单与进度记忆。
- 深色主题 + 车机横屏适配。

### V1.1（第二阶段）
- 驾驶模式全屏界面。
- 最近播放自动生成。
- 歌曲封面读取与显示。
- 身份默认颜色库。

### V1.2（第三阶段）
- 方向盘按键/快捷键绑定。
- 语音控制接口预留。
- 多设备播放状态同步（可选，依赖懒猫账号体系）。
- 在线歌词/封面下载（当前暂不考虑，需 `net.internet` 权限）。

---

## 十、风险与约束

| 风险 | 影响 | 应对 |
|---|---|---|
| 车机浏览器 AudioContext 自动播放受限 | 高 | 首次交互后再启动音频；提供显式“开始播放”按钮。 |
| 懒猫网盘大目录扫描慢 | 中 | 后端异步扫描 + 缓存索引；前端先展示已有数据。 |
| 不同车机屏幕分辨率差异大 | 中 | 采用响应式布局，横屏优先，支持常见 7~12 英寸车机。 |
| 音频格式支持不一 | 中 | 优先支持 MP3/FLAC/AAC/M4A，必要时后端转码。 |
| 多身份数据隔离与存储容量 | 低 | 元数据很小；歌曲文件只存引用不复制。 |
| USB/SMB 存储拔出后歌曲不可用 | 中 | 播放时检测文件是否存在，不存在则跳过并提示；恢复播放时若最后歌曲不可用，从歌单第一首开始。 |

---

## 十一、待确认事项

- [x] 最终确定应用名称：中文 **多音盒**，英文 **MultiTune**，包名 `ink.akawa.ety001.multitune`。
- [x] 后端服务：使用 **Go + Gin + SQLite**。
- [x] 前端方案：双版本（Vue 3 现代版 + 纯 ES5 简化版），兼容比亚迪等老版本车机浏览器。
- [x] 音乐文件来源：通过多个 `binds` 把懒猫主目录、USB 存储、SMB 共享等挂载到容器内任意路径，歌单保存文件路径索引。
- [x] 车机兼容：以比亚迪低版本内核为基准，简化版前端兼容 Chrome/WebView ≤ 74，默认覆盖其他车机。
- [x] 在线功能：歌词、封面下载暂不需要。
- [x] 音量控制：需要，UI 提供音量滑块。
- [ ] 是否接入懒猫通知/语音等扩展能力？
- [ ] 存储挂载路径：初期开发不绑定，LPK 打包阶段在 `lzc-manifest.yml` 中按需配置。

---

## 十二、附录

### 12.1 包名（懒猫 LPK package ID）

```
ink.akawa.ety001.multitune
```

### 12.2 目录规划（建议）
```
~/workspace/lzc-appdb/multitune/        # LPK 打包配置
  ├── package.yml
  ├── lzc-manifest.yml
  ├── lzc-build.yml
  └── icon.png

~/workspace/multitune/                  # 应用源码
  ├── backend/                          # Go + Gin + SQLite
  │   ├── cmd/server/main.go
  │   ├── internal/
  │   │   ├── api/                      # HTTP handlers
  │   │   ├── db/                       # SQLite 封装与迁移
  │   │   ├── scanner/                  # 音频扫描与元数据
  │   │   ├── stream/                   # 音频流服务
  │   │   └── model/                    # 数据模型
  │   ├── migrations/
  │   ├── go.mod
  │   ├── go.sum
  │   └── Dockerfile
  ├── frontend/                         # Vue 3 现代版
  │   ├── src/
  │   ├── index.html
  │   ├── package.json
  │   └── vite.config.ts
  ├── simple/                           # 老车机简化版（纯 ES5 JS）
  │   ├── index.html
  │   ├── identity.html
  │   ├── playlist.html
  │   └── player.html
  └── docker/Dockerfile                 # 总构建 Dockerfile
```

---

*下一步建议：确认音乐文件来源与车机兼容范围，随后进入后端 API 详细设计、数据库迁移脚本与简化版前端原型搭建。*
