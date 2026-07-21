# AGENTS.md — 多音盒 MultiTune 开发规范

> 本文件供 AI coding agent 在创建/修改代码前强制读取并遵守。

## 通用规则

- 语言：Go 1.23+，前端 Vue 3 / 纯 ES5 双版本。
- 包管理：前端用 pnpm，不要用 npm。
- 静态编译：CGO_ENABLED=0（使用 modernc.org/sqlite，CGO-free）。
- 提交前必须通过：`go vet ./...` + `gofmt -l .`（无输出）+ `go test -race ./...`。

## 后端编码规范（审计强制要求）

以下规则在每个 PR 中必须遵守，审计会逐条检查。违反任何一条都会被打回。

### 1. slog.Error 必填

每个 handler 中的 `http.StatusInternalServerError` 分支，必须加 `slog.Error` 记录原始错误。

错误示例（禁止）：

```go
if err != nil {
    c.JSON(http.StatusInternalServerError, model.APIResponse{
        Code: 9001, Message: "内部错误",
    })
    return
}
```

正确示例：

```go
if err != nil {
    slog.Error("查询身份失败", "error", err, "id", id)
    c.JSON(http.StatusInternalServerError, model.APIResponse{
        Code: 9001, Message: "内部错误",
    })
    return
}
```

没有例外。每个 500 分支都要有日志。

### 2. List 方法返回空 slice 而非 nil

所有返回列表的 repository 方法，用 `make([]T, 0)` 初始化，不要用 `var xxx []T`。

原因：nil slice 的 JSON 序列化结果为 `null`，前端需要额外判空；空 slice 序列化为 `[]`。

错误示例（禁止）：

```go
var songs []model.Song
```

正确示例：

```go
songs := make([]model.Song, 0)
```

### 3. Update 操作禁止 read-modify-write

不要先 `GetByID` 读到内存、修改字段、再 `UPDATE` 全量写入。这是并发竞态。

错误示例（禁止）：

```go
func (r *Repo) Update(id string, name *string) (*Model, error) {
    existing, _ := r.GetByID(id)  // 读
    if name != nil {
        existing.Name = *name     // 改
    }
    r.db.Exec(`UPDATE ... SET name = ?`, existing.Name)  // 写全量
    return existing, nil
}
```

正确示例（动态 SET 子句，只更新传入字段）：

```go
func (r *Repo) Update(id string, name *string) (*Model, error) {
    existing, err := r.GetByID(id)
    if err != nil {
        return nil, err
    }
    if existing == nil {
        return nil, nil
    }

    setParts := []string{"updated_at = ?"}
    args := []interface{}{time.Now().Unix()}
    if name != nil {
        setParts = append(setParts, "name = ?")
        args = append(args, *name)
    }
    args = append(args, id)

    query := "UPDATE table SET " + strings.Join(setParts, ", ") + " WHERE id = ?"
    if _, err := r.db.Exec(query, args...); err != nil {
        return nil, fmt.Errorf("更新失败: %w", err)
    }

    return r.GetByID(id) // 返回最新数据
}
```

### 4. Delete 操作必须区分存在/不存在

repo 的 Delete 方法返回 `(bool, error)` 或等价机制。handler 对不存在的资源返回 404。

repo 示例：

```go
func (r *Repo) Delete(id string) (bool, error) {
    result, err := r.db.Exec(`DELETE FROM xxx WHERE id = ?`, id)
    if err != nil {
        return false, fmt.Errorf("删除失败: %w", err)
    }
    rows, _ := result.RowsAffected()
    return rows > 0, nil
}
```

handler 示例：

```go
deleted, err := h.repo.Delete(id)
if err != nil { /* slog.Error + 500 */ }
if !deleted {
    c.JSON(http.StatusNotFound, ...)
    return
}
```

### 5. 业务错误用 sentinel error

repo 层的业务错误（如"歌曲不在歌单中"、"身份不存在"）用 sentinel error 定义，handler 用 `errors.Is` 判断，返回正确的 HTTP 状态码。

repo 定义：

```go
var ErrSongNotInPlaylist = errors.New("歌曲不在歌单中")

func (r *Repo) UpdateSongOrder(...) error {
    if rows == 0 {
        return fmt.Errorf("%w: %s", ErrSongNotInPlaylist, songID)
    }
}
```

handler 使用：

```go
if err := h.repo.UpdateSongOrder(...); err != nil {
    if errors.Is(err, repository.ErrSongNotInPlaylist) {
        c.JSON(http.StatusBadRequest, ...)  // 400，不是 500
        return
    }
    slog.Error("更新失败", "error", err)
    c.JSON(http.StatusInternalServerError, ...)
}
```

### 6. 不用字符串比较判断错误类型

不要用 `err.Error() == "xxx"` 做逻辑分支。用 sentinel error 或自定义错误类型 + `errors.Is` / `errors.As`。标准库错误同理（如 `io.EOF`）。

错误示例（禁止）：

```go
if err.Error() == "路径不存在" {
    c.JSON(http.StatusBadRequest, ...)
}

_, err = f.Readdirnames(1)
return err == nil || err.Error() == "EOF"
```

正确示例：

```go
if errors.Is(err, fsutil.ErrPathNotFound) {
    c.JSON(http.StatusBadRequest, ...)
}

_, err = f.Readdirnames(1)
return err == nil || errors.Is(err, io.EOF)
```

### 7. Upsert 用原子操作

不要先 SELECT 再判断 INSERT 或 UPDATE（read-modify-write 竞态）。用 `INSERT OR IGNORE` + 后续 UPDATE，或 `INSERT ... ON CONFLICT DO UPDATE`。

**部分字段合并（partial upsert）也必须原子**：当"未传入的字段保留数据库现有值"时，不要在 handler 里先 GET 读现有值、内存合并、再全量 upsert——两个客户端并发部分更新会互相覆盖。把合并逻辑放进单条 SQL，用 `CASE WHEN` 按"字段是否传入"的布尔参数选择新值或旧值：

```go
// 指针参数为 nil 表示未传入、保留现有值
_, err = tx.Exec(`
    INSERT INTO playback_states (identity_id, playlist_id, song_id, position, mode, updated_at)
    VALUES (?, ?, ?, ?, ?, ?)
    ON CONFLICT(identity_id) DO UPDATE SET
        playlist_id = CASE WHEN ? THEN excluded.playlist_id ELSE playback_states.playlist_id END,
        song_id     = CASE WHEN ? THEN excluded.song_id     ELSE playback_states.song_id     END,
        position    = CASE WHEN ? THEN excluded.position    ELSE playback_states.position    END,
        mode        = CASE WHEN ? THEN excluded.mode        ELSE playback_states.mode        END,
        updated_at  = excluded.updated_at
`, identityID, pl, sg, pos, md, now,
    playlistID != nil, songID != nil, position != nil, mode != nil)
```

派生表（如从播放状态同步单曲进度、歌单记忆点）要在同一事务内、从合并后的最终行 `INSERT ... SELECT ... FROM 主表 WHERE ...` 派生，不要用 handler 内存里的合并值——保证多表一致。

handler 层读现有值仅允许用于**校验**（如判断歌曲是否属于生效歌单），不允许把读到的值再写回去。

### 8. N+1 查询禁止

批量校验资源存在性时，不要用循环逐条查询。用单条 SQL 批量查询。

错误示例（禁止）：

```go
for _, id := range ids {
    song, _ := repo.GetByID(id)  // N 次 SELECT
}
```

正确示例：

```go
count, _ := repo.CountByIDs(ids)  // 1 次 SELECT
if count != len(ids) {
    // 部分不存在
}
```

### 9. 路径访问范围

应用不再锁定 `MEDIA_ROOT`，文件浏览器和扫描/流接口均可访问容器文件系统内的任意路径。
- 后端不再对路径做 `MEDIA_ROOT` 沙箱校验；
- 访问控制交给部署方通过容器挂载、反向代理或网关自行限制；
- 代码中保留对路径不存在、非目录等 IO 错误的正常处理。

### 10. Update 不应修改非目标字段

`UPDATE` 语句只修改请求中传入的字段。不要顺带更新 `created_at` 等非目标字段。

### 11. 校验跨资源关联，不只校验单个资源存在

写入涉及多个资源关联的数据时，除了逐个校验资源存在，还要校验它们的**关联关系**成立。只查 `GetByID` 不为 nil 是不够的。

例：保存播放状态时同时传入 `playlist_id` 和 `song_id`，除了校验歌单存在、歌曲存在，还必须校验**歌曲在该歌单中**，否则歌单记忆点会存入不属于该歌单的歌，前端续播时找不到。关系不成立返回 400 + 业务错误码（如 `ErrCodeSongNotInPlaylist`）。

关联校验用单条 SQL（如 `SELECT COUNT(*) FROM playlist_songs WHERE playlist_id = ? AND song_id = ?`），遵守规则 8 不做 N+1。部分更新场景下，校验对象是"合并后的生效值"（传入值与数据库现有值合并的结果），但该读取仅用于校验，写入仍走规则 7 的原子路径。

## 简化版前端开发规范

简化版面向 Chrome/WebView ≤ 74 的老车机浏览器（如比亚迪车机），必须严格遵守以下兼容性原则。PR 审计会检查简化版代码是否符合。

### F1. 只允许 ES5 语法

简化版 `web/car/` 下的 JS 必须严格使用 ES5，禁止以下特性：

- `let` / `const` → 用 `var`
- 箭头函数 `() => {}` → 用 `function`
- 模板字符串 `` `hello ${name}` `` → 用字符串拼接
- `Promise` / `async` / `await` → 用回调函数
- `Array.prototype.find` / `findIndex` / `includes` 等 ES6+ 方法 → 用 `for` 循环或 jQuery
- `String.prototype.padStart` / `padEnd` 等 ES2017 方法 → 手写辅助函数
- `Object.assign` / 展开运算符 → 用逐个字段复制
- `class` → 用构造函数 + `prototype`

允许使用：

- `JSON.stringify` / `JSON.parse`
- `XMLHttpRequest`（或 jQuery 封装的 `$.ajax`）
- 普通的 `Array.prototype.forEach` / `map` / `filter`（Chrome 74 支持，但为安全优先用手写循环）

### F2. 禁止 CSS Flexbox / Grid / CSS 变量

老车机浏览器对 Flexbox/Grid 支持不完整，必须用传统布局：

- 禁止：`display: flex`、`display: grid`、`CSS Grid` 属性
- 推荐：`display: block`、`display: inline-block`、`float`、百分比宽度、`position`
- 禁止：CSS 自定义属性（变量）`--var`
- 推荐：直接写具体颜色/尺寸值

### F3. 使用 jQuery 3.x 但保持 ES5

简化版使用本地 `jquery-3.7.1.min.js`，DOM 操作和 AJAX 用 jQuery 简化，但回调与辅助函数仍须符合 F1。

### F4. 避免依赖现代浏览器 API

- 音频播放使用 HTML5 `<audio>` 元素，不依赖 `AudioContext`（需要用户手势才能自动播放）
- 文件浏览器使用原生 `<input type="file">` 或后端 API，不依赖 `File System Access API`
- 存储使用后端 API + URL 查询参数，不依赖 `localStorage` / `IndexedDB`（如非必要）

### F5. 优先触控与大按钮

车机以触控为主，简化版交互原则：

- 主要按钮高度 ≥ 64px，点击区域足够大
- 每屏主要操作不超过 3 个
- 身份卡片尺寸 ≥ 120×120px
- 避免复杂手势，优先单击

### F6. 提交前简化版自查

- [ ] `web/car/` 下无 ES6+ 语法
- [ ] `web/car/` 下无 Flexbox/Grid/CSS 变量
- [ ] 页面在 Chrome 74 模拟器或真机上能正常打开
- [ ] 不依赖未本地化的外部资源（如 CDN 需有本地 fallback）

## 测试规范

- 每个 PR 必须包含对应测试（repo 层 + handler 层）。
- 测试用 `t.TempDir()` 创建临时数据库，每个测试独立隔离。
- 必须覆盖：成功路径 + 错误路径（不存在、参数校验、超限）。
- handler 测试用 `httptest.NewRecorder`，不启动真实 HTTP 服务。
- 运行 `go test -race -count=1 ./...` 确保无竞态。

## PR Body 格式规范

历史问题：曾多次因使用 `gh pr create --body "..."` 内联字符串导致 `\n` 被转义为字面量，PR Body 出现 `\n\n` 而格式错乱。

正确做法：

- 优先使用 `gh pr create --body-file <file>` 或 `gh pr edit <num> --body-file <file>`，将正文写入独立的 `.md` 文件。
- 如需内联，必须使用 `$'...'` 或 here-document，确保换行符为真实换行而非 `\n` 字面量。
- 创建/编辑后，用 `gh pr view <num>` 检查最终渲染效果。

## PR 提交清单

提交前自查：

- [ ] `go vet ./...` 通过
- [ ] `gofmt -l .` 无输出
- [ ] `go test -race -count=1 ./...` 通过
- [ ] 所有 500 分支有 `slog.Error`
- [ ] 所有 List 方法用 `make([]T, 0)`
- [ ] Update 操作无 read-modify-write
- [ ] 部分字段合并 upsert 用 `CASE WHEN` 原子完成，未在 handler 内存合并后写回
- [ ] Delete 操作区分存在/不存在
- [ ] 业务错误用 sentinel error + `errors.Is`（含标准库错误如 `io.EOF`，无 `err.Error() ==` 比较）
- [ ] 无 N+1 查询
- [ ] 跨资源写入校验了关联关系（不只校验单个资源存在）
- [ ] 测试覆盖成功 + 错误路径
- [ ] PR Body 使用 `--body-file` 写入，无 `\n` 字面量换行，并已用 `gh pr view` 检查

## Related project directories

本项目跨越多个目录。跨目录访问一律使用**绝对路径**，相对路径无法从不同工作目录解析。

### Registry

- `/home/ety001/workspace/lzc-appdb/multitune` — 懒猫微服（Lazycat / LZC）应用商店**打包与发布目录**（独立 git 仓库，非本工作区子目录）。包含：
    - `lzc-manifest.yml` — 应用清单：子域名 `multitune`、路由 `/=http://multitune:8080`、容器镜像 tag（`registry.lazycat.cloud/ety001/ety001/multitune:<tag>`）、挂载 `/lzcapp/var/data:/app/data`、环境变量（`LAZYCAT_DEPLOY=true` 等）。
    - `package.yml` — 应用包元信息：包名 `ink.akawa.ety001.multitune`、版本（当前 `0.0.1`）、名称、权限（`media.read`、`document.read`）、locale。
    - `lzc-build.yml` — LZC 构建配置，图标指定为 `./multitune.png`，产物输出到 `./`。
    - 构建产物 `*.lpk`（已被该目录 `.gitignore` 忽略）。

### Dependencies between directories

- **镜像 tag 同步（强）**：本工作区构建并推送容器镜像后 → 必须更新打包目录 `lzc-manifest.yml` 中 `services.multitune.image` 的 tag，否则应用商店发布的仍是旧镜像。
- **版本号同步（强）**：本工作区发布新语义版本时 → 同步更新打包目录 `package.yml` 的 `version` 字段，两者版本号应保持一致。
- **打包目录不包含应用源码**：其镜像依赖本工作区 `Dockerfile` 的构建产物；修改打包目录的清单/元信息不影响应用运行逻辑，仅影响商店发布与运行时配置（端口、挂载、环境变量）。
- **图标为独立资源（弱）**：打包目录的商店图标 `multitune.png` 与本工作区的 `web/logo.png`（应用内 UI）是**两份独立维护的资源**，文件名不同、用途不同。变更其一不强制同步另一者；若希望统一，需显式手动同步。
- **路径再校验**：注册的路径是文本记录，在进行跨目录的破坏性操作前，应先确认路径仍然存在。

### 调试工作流：把 LZC 应用路由指向本地开发环境（懒猫微服官方推荐）

懒猫微服支持一种免打 `.lpk` 的快速调试法：本地直接跑开发服务，再让懒猫设备上的应用把请求"回环"到局域网内的开发机。适用前提：开发机与懒猫设备处于**同一局域网**。

**前提条件（已验证）**：后端入口 `backend/cmd/server/main.go` 用 `addr := fmt.Sprintf(":%s", cfg.Port)` 启动，`:` 前留空即监听 `0.0.0.0`，因此 `go run`/`go build` 起来的服务**默认就对局域网开放**，无需额外配置。端口取自 `cfg.Port`（Dockerfile 默认 `PORT=8080`）。

**步骤**：

1. 在开发机本地启动后端（及前端构建/watch）。确认监听端口，默认 `8080`。
2. 取开发机局域网 IP，例如 `192.168.44.2`（用 `ip addr` 或 `ipconfig` 确认，不要用 `localhost`/`127.0.0.1`）。
3. 编辑打包目录的 `/home/ety001/workspace/lzc-appdb/multitune/lzc-manifest.yml`，把 `application.routes` 的目标从容器服务地址改为开发机地址：

   ```yaml
   # 调试态（指向本地开发环境，请勿提交）
   routes:
     - /=http://192.168.44.2:8080
   ```

   生产态原始值为 `- /=http://multitune:8080`（容器内服务名）。
4. 在懒猫设备上重装/刷新该应用，此时应用内的请求会被路由到开发机的本地服务，可直接调试。
5. **调试结束务必还原 `lzc-manifest.yml` 的 `routes` 为 `http://multitune:8080`，再提交**。调试态路由指向局域网私网地址，提交后会破坏线上分发。

**注意事项**：

- 调试态改动**只改 `routes` 目标**，不要顺手提交；该文件是发布清单，私网地址不可入库。
- 开发机防火墙需放行对应端口（如 `8080`），否则懒猫设备连不上。
- 此法调试的是「应用通过 LZC 路由访问后端」的链路；容器构建/镜像本身仍需走正常的镜像 tag 流程（见上方「镜像 tag 同步」）。
