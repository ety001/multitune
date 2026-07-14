# 多音盒 MultiTune — API 接口与数据库设计

> 文档版本：v0.1
> 关联文档：`car-music-multi-identity-prd.md`

---

## 一、通用约定

### 1.1 基础路径
所有 API 以 `/api` 为前缀。

### 1.2 响应格式

**成功响应**：
```json
{
  "code": 0,
  "message": "ok",
  "data": { ... }
}
```

**列表响应**：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "items": [ ... ],
    "total": 100
  }
}
```

**错误响应**：
```json
{
  "code": 1001,
  "message": "身份不存在"
}
```

### 1.3 HTTP 状态码
- `200`：请求成功
- `400`：请求参数错误
- `404`：资源不存在
- `500`：服务器内部错误

### 1.4 错误码定义

| 错误码 | 说明 |
|---|---|
| 0 | 成功 |
| 1001 | 身份不存在 |
| 1002 | 身份名称不能为空 |
| 1003 | 身份数量超过上限 |
| 2001 | 歌单不存在 |
| 2002 | 歌单名称不能为空 |
| 2003 | 歌单数量超过上限 |
| 3001 | 歌曲不存在 |
| 3002 | 歌曲文件不可读 |
| 4001 | 存储源不存在 |
| 4002 | 路径不存在或不可访问 |
| 5001 | 播放状态不存在 |
| 9001 | 内部错误 |

---

## 二、数据库设计

### 2.1 表结构（SQLite）

```sql
-- 身份表
CREATE TABLE identities (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    avatar_color TEXT NOT NULL DEFAULT '#6366f1',
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_default INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- 歌单表
CREATE TABLE playlists (
    id TEXT PRIMARY KEY,
    identity_id TEXT NOT NULL,
    name TEXT NOT NULL,
    cover_url TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (identity_id) REFERENCES identities(id) ON DELETE CASCADE
);

-- 歌曲表（文件索引）
CREATE TABLE songs (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    source TEXT NOT NULL DEFAULT 'unknown',
    title TEXT NOT NULL,
    artist TEXT,
    album TEXT,
    duration INTEGER NOT NULL DEFAULT 0,
    cover_url TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- 歌单-歌曲关联表
CREATE TABLE playlist_songs (
    playlist_id TEXT NOT NULL,
    song_id TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    PRIMARY KEY (playlist_id, song_id),
    FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
    FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE
);

-- 播放状态表（每个身份一条）
CREATE TABLE playback_states (
    identity_id TEXT PRIMARY KEY,
    playlist_id TEXT,
    song_id TEXT,
    position INTEGER NOT NULL DEFAULT 0,
    mode TEXT NOT NULL DEFAULT 'order',
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (identity_id) REFERENCES identities(id) ON DELETE CASCADE,
    FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE SET NULL,
    FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE SET NULL
);

-- 索引
CREATE INDEX idx_playlists_identity_id ON playlists(identity_id);
CREATE INDEX idx_playlist_songs_playlist_id ON playlist_songs(playlist_id);
CREATE INDEX idx_playlist_songs_song_id ON playlist_songs(song_id);
CREATE INDEX idx_songs_path ON songs(path);
CREATE INDEX idx_songs_source ON songs(source);
```

### 2.2 字段说明

#### identities
| 字段 | 类型 | 说明 |
|---|---|---|
| id | TEXT | UUID，主键 |
| name | TEXT | 身份名称，如“爸爸”、“通勤” |
| avatar_color | TEXT | 身份卡片颜色，HEX 格式 |
| sort_order | INTEGER | 排序权重 |
| is_default | INTEGER | 0/1，是否为上车默认身份 |
| created_at | INTEGER | 创建时间戳（秒） |
| updated_at | INTEGER | 更新时间戳（秒） |

#### playlists
| 字段 | 类型 | 说明 |
|---|---|---|
| id | TEXT | UUID，主键 |
| identity_id | TEXT | 所属身份 ID |
| name | TEXT | 歌单名称 |
| cover_url | TEXT | 歌单封面 URL（可选） |
| sort_order | INTEGER | 排序权重 |
| created_at | INTEGER | 创建时间戳 |
| updated_at | INTEGER | 更新时间戳 |

#### songs
| 字段 | 类型 | 说明 |
|---|---|---|
| id | TEXT | UUID，主键 |
| path | TEXT | 容器内绝对路径，唯一 |
| source | TEXT | 来源标识：home / usb / smb / unknown |
| title | TEXT | 歌曲标题（优先读取 ID3，fallback 文件名） |
| artist | TEXT | 艺术家 |
| album | TEXT | 专辑 |
| duration | INTEGER | 时长（秒） |
| cover_url | TEXT | 封面 URL（可选） |
| created_at | INTEGER | 创建时间戳 |
| updated_at | INTEGER | 更新时间戳 |

#### playlist_songs
| 字段 | 类型 | 说明 |
|---|---|---|
| playlist_id | TEXT | 歌单 ID |
| song_id | TEXT | 歌曲 ID |
| sort_order | INTEGER | 歌曲在歌单中的排序 |
| created_at | INTEGER | 添加时间戳 |

#### playback_states
| 字段 | 类型 | 说明 |
|---|---|---|
| identity_id | TEXT | 身份 ID，主键 |
| playlist_id | TEXT | 最后播放的歌单 ID |
| song_id | TEXT | 最后播放的歌曲 ID |
| position | INTEGER | 歌曲播放位置（秒） |
| mode | TEXT | 播放模式：order / random / single-loop |
| updated_at | INTEGER | 更新时间戳 |

### 2.3 数据约束

- 一个身份下最多 N 个歌单（默认 50，可通过配置调整）。
- 一个歌单中最多 N 首歌曲（默认 1000）。
- `songs.path` 唯一，避免同一文件重复入库。
- 删除身份时，级联删除其歌单、歌单-歌曲关联、播放状态。
- 删除歌单时，只删除关联，不删除 `songs` 表记录。
- 删除歌曲时，级联删除歌单-歌曲关联，播放状态中的 song_id 设为 NULL。

---

## 三、API 接口详细定义

### 3.1 身份 API

#### GET /api/identities
获取身份列表。

**响应**：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "items": [
      {
        "id": "uuid-1",
        "name": "爸爸",
        "avatar_color": "#6366f1",
        "sort_order": 0,
        "is_default": 1,
        "created_at": 1720934400,
        "updated_at": 1720934400
      }
    ],
    "total": 1
  }
}
```

#### POST /api/identities
创建身份。

**请求体**：
```json
{
  "name": "妈妈",
  "avatar_color": "#ec4899"
}
```

**响应**：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "uuid-2",
    "name": "妈妈",
    "avatar_color": "#ec4899",
    "sort_order": 1,
    "is_default": 0,
    "created_at": 1720934400,
    "updated_at": 1720934400
  }
}
```

#### GET /api/identities/:id
获取身份详情（含歌单列表）。

**响应**：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "uuid-1",
    "name": "爸爸",
    "avatar_color": "#6366f1",
    "sort_order": 0,
    "is_default": 1,
    "playlists": [
      {
        "id": "pl-1",
        "name": "通勤",
        "song_count": 12
      }
    ],
    "created_at": 1720934400,
    "updated_at": 1720934400
  }
}
```

#### PUT /api/identities/:id
更新身份。

**请求体**：
```json
{
  "name": "爸爸-改",
  "avatar_color": "#10b981",
  "sort_order": 2
}
```

**说明**：字段均可选，传入什么更新什么。

#### DELETE /api/identities/:id
删除身份。

**响应**：
```json
{
  "code": 0,
  "message": "ok",
  "data": null
}
```

#### POST /api/identities/:id/default
设为默认身份。

**说明**：
- 一个应用内只能有一个默认身份。
- 设置某身份为默认时，自动将其他身份的 `is_default` 置为 0。
- 创建第一个身份时后端自动将其设为默认。

**响应**：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "uuid-1",
    "is_default": 1
  }
}
```

#### DELETE /api/identities/:id 的默认身份处理
- 如果删除的是默认身份，且还有其他身份，自动将 `sort_order` 最小的身份设为默认。
- 如果删除后没有身份，应用进入首次使用引导状态。

---

### 3.2 歌单 API

#### GET /api/identities/:id/playlists
获取身份下的歌单列表。

**响应**：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "items": [
      {
        "id": "pl-1",
        "identity_id": "uuid-1",
        "name": "通勤",
        "cover_url": null,
        "song_count": 12,
        "sort_order": 0,
        "created_at": 1720934400,
        "updated_at": 1720934400
      }
    ],
    "total": 1
  }
}
```

#### POST /api/playlists
创建歌单。

**请求体**：
```json
{
  "identity_id": "uuid-1",
  "name": "跑山",
  "sort_order": 1
}
```

**响应**：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "pl-2",
    "identity_id": "uuid-1",
    "name": "跑山",
    "cover_url": null,
    "sort_order": 1,
    "created_at": 1720934400,
    "updated_at": 1720934400
  }
}
```

#### GET /api/playlists/:id
获取歌单详情（含歌曲列表）。

**响应**：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "pl-1",
    "identity_id": "uuid-1",
    "name": "通勤",
    "cover_url": null,
    "sort_order": 0,
    "songs": [
      {
        "id": "song-1",
        "path": "/app/media/home/music/xxx.mp3",
        "source": "home",
        "title": "歌曲名",
        "artist": "歌手",
        "album": "专辑",
        "duration": 180,
        "cover_url": null,
        "sort_order": 0
      }
    ],
    "created_at": 1720934400,
    "updated_at": 1720934400
  }
}
```

#### PUT /api/playlists/:id
更新歌单。

**请求体**：
```json
{
  "name": "通勤-改",
  "sort_order": 2
}
```

#### DELETE /api/playlists/:id
删除歌单。

#### POST /api/playlists/:id/songs
添加歌曲到歌单。

**请求体**：
```json
{
  "song_ids": ["song-1", "song-2"]
}
```

**说明**：
- `song_ids` 必须已存在于 `songs` 表中。
- 如需将未扫描的文件加入歌单，先调用 `POST /api/scan` 扫描目录或文件，再使用返回的 `song_id` 调用本接口。
- 已存在的歌曲不重复添加。
- 追加到歌单末尾。

#### DELETE /api/playlists/:id/songs/:songId
从歌单移除歌曲。

#### PUT /api/playlists/:id/songs/order
调整歌单内歌曲顺序。

**请求体**：
```json
{
  "song_ids": ["song-2", "song-1", "song-3"]
}
```

---

### 3.3 歌曲与扫描 API

#### POST /api/scan
扫描指定目录或文件，返回发现的歌曲。

**请求体**：
```json
{
  "path": "/app/media/home/music"
}
```

或扫描单个文件：
```json
{
  "path": "/app/media/home/music/xxx.mp3"
}
```

**响应**：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "scanned": 50,
    "added": 10,
    "songs": [
      {
        "id": "song-1",
        "path": "/app/media/home/music/xxx.mp3",
        "source": "home",
        "title": "歌曲名",
        "artist": "歌手",
        "album": "专辑",
        "duration": 180
      }
    ]
  }
}
```

**说明**：
- 扫描是幂等的，同一目录多次扫描不会重复入库。
- 扫描只发现支持的音频格式：mp3, flac, m4a, aac, ogg, wav。
- 扫描过程异步或同步均可，目录大时建议返回任务 ID（V1 可简化为同步）。

#### GET /api/songs
歌曲列表/搜索。

**查询参数**：
- `q`: 搜索关键词（标题/艺术家/专辑）
- `source`: 按来源过滤
- `limit`: 默认 20
- `offset`: 默认 0

**响应**：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "items": [ ... ],
    "total": 100
  }
}
```

#### GET /api/songs/:id
歌曲详情。

#### GET /api/songs/:id/cover
歌曲封面。

**说明**：
- 如果歌曲文件内嵌封面，直接提取返回。
- 如果没有内嵌封面，返回默认封面图片。
- 响应 `Content-Type: image/jpeg` 或 `image/png`。

#### GET /api/stream?songId=xxx
音频流。

**说明**：
- 支持 HTTP Range 请求。
- 根据文件扩展名返回正确 MIME 类型。
- 现代版和简化版播放器均使用该接口播放。

---

### 3.4 文件浏览器 API

#### GET /api/fs/sources
获取已挂载的存储源列表。

**响应**：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "items": [
      { "id": "home", "name": "主目录", "path": "/app/media/home" },
      { "id": "usb", "name": "USB 存储", "path": "/app/media/usb" },
      { "id": "smb", "name": "SMB 共享", "path": "/app/media/smb" }
    ]
  }
}
```

#### GET /api/fs/list?path=xxx
列出指定路径下的文件和文件夹。

**查询参数**：
- `path`: 容器内绝对路径，如 `/app/media/home/music`

**响应**：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "path": "/app/media/home/music",
    "parent": "/app/media/home",
    "items": [
      {
        "name": "pop",
        "type": "dir",
        "path": "/app/media/home/music/pop"
      },
      {
        "name": "xxx.mp3",
        "type": "file",
        "path": "/app/media/home/music/xxx.mp3",
        "is_audio": true,
        "size": 5242880
      }
    ]
  }
}
```

#### GET /api/fs/search?q=xxx&source=home
跨存储源搜索歌曲。

**查询参数**：
- `q`: 搜索关键词
- `source`: 可选，指定来源

**说明**：
- 搜索基于已扫描入库的 `songs` 表，不实时扫描文件系统。
- 返回结果包含歌曲完整信息。

---

### 3.5 播放状态 API

#### GET /api/playback/:identityId
获取身份播放状态。

**响应**：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "identity_id": "uuid-1",
    "playlist_id": "pl-1",
    "song_id": "song-1",
    "position": 125,
    "mode": "order",
    "updated_at": 1720934400
  }
}
```

**说明**：
- 如果该身份没有播放状态，返回空对象或默认状态。

#### POST /api/playback/:identityId
保存/更新播放状态。

**请求体**：
```json
{
  "playlist_id": "pl-1",
  "song_id": "song-1",
  "position": 125,
  "mode": "order"
}
```

**说明**：
- 字段均可选，未传入字段保持原值。
- 前端播放过程中定时（如每 5 秒）调用该接口保存进度。
- 切换歌曲、暂停时也应保存一次。

---

### 3.6 通用 API

#### GET /api/healthz
健康检查。

**响应**：
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "status": "ok"
  }
}
```

#### POST /api/device-info
接收并记录车机设备信息（参考 lzc-story）。

**请求体**：
```json
{
  "userAgent": "...",
  "chromeVersion": 74,
  "webViewVersion": 74,
  "isWebView": true,
  "screenWidth": 1920,
  "screenHeight": 1080,
  "windowWidth": 1920,
  "windowHeight": 1080,
  "language": "zh-CN",
  "platform": "Linux armv8l",
  "cookieEnabled": true,
  "onLine": true,
  "timestamp": "2026-07-14T10:00:00Z"
}
```

**说明**：
- 用于排查车机兼容问题。
- 数据可写入日志或单独的 `device_logs` 表（可选）。

---

## 四、配置与环境变量

| 变量名 | 默认值 | 说明 |
|---|---|---|
| `PORT` | `8080` | HTTP 服务端口 |
| `DATA_PATH` | `/app/data` | SQLite 数据库与封面缓存目录 |
| `MEDIA_ROOT` | `/app/media` | 音乐文件挂载根目录 |
| `DATABASE_NAME` | `multitune.db` | SQLite 数据库文件名 |
| `MAX_IDENTITIES` | `20` | 最大身份数量 |
| `MAX_PLAYLISTS_PER_IDENTITY` | `50` | 每个身份最大歌单数 |
| `MAX_SONGS_PER_PLAYLIST` | `1000` | 每个歌单最大歌曲数 |
| `SCAN_FORMATS` | `mp3,flac,m4a,aac,ogg,wav` | 扫描支持的音频格式 |
| `PLAYBACK_SAVE_INTERVAL` | `5` | 播放进度自动保存间隔（秒） |
| `LOG_LEVEL` | `info` | 日志级别：debug/info/warn/error |

## 五、数据库迁移

采用手写迁移脚本管理表结构：

```
backend/migrations/
  ├── 001_init_schema.sql
  ├── 002_add_indexes.sql
  └── ...
```

迁移记录表：
```sql
CREATE TABLE IF NOT EXISTS _migrations (
    version INTEGER PRIMARY KEY,
    applied_at INTEGER NOT NULL
);
```

启动时按版本号顺序执行：
1. 读取 `_migrations` 表记录当前版本。
2. 对比 migrations 目录中的脚本，执行未执行的脚本。
3. 每执行一条，写入 `_migrations` 表。

要求：
- 新增字段必须加默认值，保证旧数据兼容。
- 删除字段/表先标记废弃，至少保留一个版本后再删除。

## 六、安全设计

### 6.1 路径校验
- `/api/fs/list`、`/api/scan`、`/api/stream` 必须校验路径在 `MEDIA_ROOT` 下，禁止访问容器其他目录。
- 拒绝包含 `..`、软链接跳出 `MEDIA_ROOT` 的路径。

### 6.2 音频流安全
- `/api/stream` 通过 `song_id` 查询 `songs.path`，不直接接受客户端路径。
- 返回前再次校验文件在 `MEDIA_ROOT` 下且可读。

### 6.3 身份隔离
- 所有按身份查询的接口必须校验 `identity_id` 存在。
- 播放状态按身份隔离，不允许跨身份读取。

### 6.4 无用户系统声明
多音盒不内置认证，部署在公网或共享环境时，需由搭建者自行配置外部访问控制（见 README.md）。

## 七、文件扫描与索引流程

### 4.1 扫描触发时机
1. 用户在文件浏览器中选择文件夹并点击"扫描"。
2. 添加歌曲到歌单时，如果歌曲不在 `songs` 表中，自动扫描单个文件元数据。

### 4.2 扫描流程
```
1. 校验路径是否在 /app/media/ 下
2. 递归遍历目录
3. 识别支持的音频格式
4. 读取 ID3 标签（标题/艺术家/专辑/封面）
5. 计算歌曲时长
6. 插入或更新 songs 表
7. 返回扫描结果
```

### 4.3 元数据读取
- **优先**：使用 Go 音频元数据库（如 `github.com/bogem/id3v2`、`github.com/mikkyang/id3-go`）读取 ID3。
- **fallback**：如果无 ID3 标签，标题使用文件名（去掉扩展名）。
- **时长**：使用 `github.com/tcolgate/mp3` 或调用 `ffprobe`。
- **封面**：内嵌封面提取后缓存到 `/app/data/covers/` 下，按 song_id 命名。

### 4.4 封面缓存
```
/app/data/
  ├── multitune.db
  ├── covers/
  │     ├── song-1.jpg
  │     ├── song-2.png
  │     └── ...
  └── backup/
```

---

## 八、播放状态保存策略

### 8.1 保存时机
- 播放过程中每 5 秒保存一次 position。
- 切换歌曲时保存上一首的 position。
- 用户暂停时保存 position。
- 用户切换身份时保存当前身份的 position。
- 页面可见性变化（visibilitychange）时保存。

### 8.2 恢复逻辑
- 进入身份时，查询该身份的 playback_state。
- 如果存在且歌曲文件可访问，从 position 位置开始播放。
- 如果歌曲文件已不可用（如 USB 拔出），从该歌单第一首开始播放。

### 8.2.1 最近播放记录（P2）
- 每次播放歌曲时，将 `song_id` 和 `played_at` 写入 `recent_plays` 表（V1.1 实现）。
- 表结构：`(song_id TEXT, identity_id TEXT, played_at INTEGER)`。
- 首页可按身份展示最近播放的歌曲列表。

### 8.3 数据库写入优化
- 播放进度保存可使用内存缓存 + 定期刷盘，避免频繁写 SQLite。
- 简单实现：每次保存直接 UPDATE，SQLite WAL 模式下性能足够（1000 次/秒以上）。

---

## 九、简化版前端 API 调用说明

简化版前端使用原生 `XMLHttpRequest` 调用上述 API：

```javascript
var xhr = new XMLHttpRequest();
xhr.open('GET', '/api/identities', true);
xhr.onreadystatechange = function() {
  if (xhr.readyState === 4 && xhr.status === 200) {
    var resp = JSON.parse(xhr.responseText);
    if (resp.code === 0) {
      var identities = resp.data.items;
      // render
    }
  }
};
xhr.send();
```

简化版前端主要调用以下接口：
- `GET /api/identities`
- `GET /api/identities/:id/playlists`
- `GET /api/playlists/:id`
- `GET /api/playback/:identityId`
- `POST /api/playback/:identityId`
- `GET /api/stream?songId=xxx`
- `GET /api/fs/sources`
- `GET /api/fs/list?path=xxx`

简化版不调用 PUT/DELETE，相关管理操作（编辑、删除）在车机场景下引导至 PC 完整版完成。

---

## 十、下一步

1. 确认 API 定义是否有遗漏或调整。
2. 确认数据库字段是否满足需求。
3. 开始编写 Go 后端骨架（main.go + Gin 路由 + SQLite 初始化）。
4. 同步设计简化版前端页面原型。
