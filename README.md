# 多音盒 MultiTune

一个面向车机场景的多身份音乐播放器，专为懒猫微服 LPK 环境设计。

## 简介

多音盒解决车内多人用车时的音乐切换痛点：每个使用者可以拥有独立的身份，每个身份下可管理多个歌单，播放进度、播放模式等状态按身份隔离。上车后一键切换身份即可继续上次的音乐。

## 核心功能

- **多身份快速切换**：自定义身份名称和颜色，车机首页大卡片一触切换。
- **按身份隔离歌单**：每个身份拥有独立歌单列表与播放记忆。
- **文件索引式歌单**：不复制音乐文件，歌单仅保存文件路径索引。
- **多存储源支持**：可同时访问懒猫主目录、USB 存储、SMB 共享等挂载目录。
- **双前端版本**：
  - 现代版：Vue 3，面向新版浏览器。
  - 简化版：jQuery + ES5，兼容 Chrome/WebView ≤ 74（如比亚迪车机）。
- **极简播放器**：播放/暂停、上下曲、进度条、播放列表、顺序/随机/单曲循环。
- **播放记忆**：记住每个身份最后播放的歌单、歌曲与进度。

## 技术栈

- **后端**：Go 1.23+、Gin、SQLite（modernc.org/sqlite）
- **现代前端**：Vue 3、Vite、Pinia
- **简化前端**：jQuery + HTML + ES5
- **部署**：Docker、懒猫微服 LPK

## 快速开始

```bash
cd ~/workspace/multitune

# 后端（指定静态文件目录为 web/）
cd backend
go mod download
STATIC_PATH=../web go run ./cmd/server

# 现代前端（开发模式）
cd ../frontend
pnpm install
pnpm dev

# 简化前端为纯静态页面，已放在 web/simple/，由后端统一托管
```

入口页 `web/index.html` 会自动检测浏览器版本：
- Chrome/WebView ≤ 74 跳转至 `/simple/`
- 其他现代浏览器跳转至 `/modern/`
- 可通过 `?v=simple` 或 `?v=modern` 强制指定版本

## ⚠️ 重要提示

**本应用不设计用户系统、登录认证或权限管理。**

多音盒定位为家庭/私有网络环境下的车机音乐播放器，应用本身不内置账号体系和访问控制。若部署在公网或多人共享且需要隔离访问的场景，**搭建者需自行配置外部保护措施**，例如：

- 通过懒猫微服网关或反向代理配置访问控制；
- 使用防火墙、VPN 等网络层隔离；
- 在需要时自行增加认证中间件。

## 项目结构

```
multitune/
├── backend/            # Go + Gin + SQLite 后端
├── frontend/           # Vue 3 现代版前端源码（构建后输出到 web/modern/）
├── web/                # 静态文件目录，由后端统一托管
│   ├── index.html      # 入口检测页
│   ├── simple/         # 简化版前端（jQuery + ES5）
│   └── modern/         # 现代版前端构建产物
├── docs/               # 项目文档
│   ├── car-music-multi-identity-prd.md
│   └── multitune-api-db-design.md
├── docker/             # Dockerfile 与构建脚本
└── README.md
```

## 文档

- [需求文档](docs/car-music-multi-identity-prd.md)
- [API 与数据库设计](docs/multitune-api-db-design.md)

## 许可证

MIT
