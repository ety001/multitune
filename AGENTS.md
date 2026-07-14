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

不要用 `err.Error() == "xxx"` 做逻辑分支。用 sentinel error 或自定义错误类型 + `errors.Is` / `errors.As`。

错误示例（禁止）：

```go
if err.Error() == "路径不存在" {
    c.JSON(http.StatusBadRequest, ...)
}
```

正确示例：

```go
if errors.Is(err, fsutil.ErrPathNotFound) {
    c.JSON(http.StatusBadRequest, ...)
}
```

### 7. Upsert 用原子操作

不要先 SELECT 再判断 INSERT 或 UPDATE（read-modify-write 竞态）。用 `INSERT OR IGNORE` + 后续 UPDATE，或 `INSERT ... ON CONFLICT DO UPDATE`。

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

### 9. 路径安全校验

所有接受用户输入路径的接口，必须用 `fsutil.ValidateMediaPath` 校验。该函数使用 `filepath.EvalSymlinks` 逐级解析软链接，防止通过软链接跳出 `MEDIA_ROOT`。

### 10. Update 不应修改非目标字段

`UPDATE` 语句只修改请求中传入的字段。不要顺带更新 `created_at` 等非目标字段。

## 测试规范

- 每个 PR 必须包含对应测试（repo 层 + handler 层）。
- 测试用 `t.TempDir()` 创建临时数据库，每个测试独立隔离。
- 必须覆盖：成功路径 + 错误路径（不存在、参数校验、超限）。
- handler 测试用 `httptest.NewRecorder`，不启动真实 HTTP 服务。
- 运行 `go test -race -count=1 ./...` 确保无竞态。

## PR 提交清单

提交前自查：

- [ ] `go vet ./...` 通过
- [ ] `gofmt -l .` 无输出
- [ ] `go test -race -count=1 ./...` 通过
- [ ] 所有 500 分支有 `slog.Error`
- [ ] 所有 List 方法用 `make([]T, 0)`
- [ ] Update 操作无 read-modify-write
- [ ] Delete 操作区分存在/不存在
- [ ] 业务错误用 sentinel error + `errors.Is`
- [ ] 无 N+1 查询
- [ ] 测试覆盖成功 + 错误路径
