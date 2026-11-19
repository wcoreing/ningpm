# ningpm

轻量项目管理：项目、成员、迭代、看板 / 列表 / 树形 / 排期、子事项、评论、附件、通知。

界面为 Vue 3，接口为 Go（Gin）。本地默认 SQLite；生产用 MySQL。

当前版本见根目录 `VERSION`。

## 结构

```
apps/web          Vue 前端（Vite）
apps/api          Go API
packages/ui       共享 UI
deploy            生产仅起 MySQL 的 compose
scripts           本地启动与部署脚本
```

前端开发时把 `/api` 代理到 `127.0.0.1:8080`。

## 本地开发

需要 Go、Node、pnpm。

```bash
pnpm install
pnpm dev:api    # API，默认 http://127.0.0.1:8080
pnpm dev        # 前端
```

空库首次启动会写入演示管理员：

- 账号：`admin@ningpm.local`
- 密码：`admin123`

仅用于本地演示，对外环境请改掉密码和 `NINGPM_JWT_SECRET`。

## 环境变量（API）

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `NINGPM_HTTP_ADDR` | `:8080` | 监听地址 |
| `NINGPM_DB_DRIVER` | `sqlite` | `sqlite` 或 `mysql` |
| `NINGPM_DB_PATH` | `data/ningpm.db` | SQLite 文件 |
| `NINGPM_DB_DSN` | 空 | MySQL DSN |
| `NINGPM_FILES_DIR` | 库旁 `files/` | 附件目录 |
| `NINGPM_JWT_SECRET` | 开发占位 | 令牌密钥，生产必须更换 |
| `NINGPM_VERSION` | `VERSION` 文件 | 响应里的版本号 |

MySQL 示例：

```bash
export NINGPM_DB_DRIVER=mysql
export NINGPM_DB_DSN='ningpm:password@tcp(127.0.0.1:3308)/ningpm?parseTime=true&charset=utf8mb4'
```

`deploy/docker-compose.yml` 只起数据库（宿主机 `127.0.0.1:3308`）。API 与前端在宿主机编译运行。

## 构建

```bash
pnpm build
cd apps/api && go build -o bin/ningpm-api ./cmd/server
```

前端产物在 `apps/web/dist`。
