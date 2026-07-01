# 把 MCP Server 改造成 Skill 的设计与实现

本文记录把本项目(小宇宙 MCP Server)扩展出 Skill 能力的完整思路、方案选型与落地过程，供后续维护参考。

## 1. MCP 与 Skill 的本质区别

| 维度 | MCP Server | Skill |
|---|---|---|
| 本质 | 常驻进程，通过协议暴露结构化工具(带 JSON Schema 参数校验) | 一个文件夹：`SKILL.md`(说明书) + 可选脚本，模型读说明后用 Bash 调用 |
| 调用方式 | 客户端按 schema 结构化调用，参数强校验 | 模型读自然语言说明，自行拼命令行执行 |
| 加载时机 | 启动即注册全部工具，常驻上下文 | 按需加载，仅相关时进上下文，更省 token |
| 可移植性 | 任何支持 MCP 的客户端通用(Claude Desktop、Cursor…) | 仅在支持 Skill 的环境可用(如 Claude Code) |

结论：两者是不同机制，不存在「直接转换」。所谓「改成 Skill」实为**用 Skill 机制重新表达同一能力**。

## 2. 为什么本项目适合改造

关键在于**鉴权不依赖 MCP**：

- 登录逻辑是独立子命令 `init`(手机号+验证码)，token 存于 `~/.mcp/xiaoyuzhoufm-mcp/token.json`，自动刷新。
- 见 `internal/xyzclient/token.go`、`cmd/xiaoyuzhoufm-mcp/main.go`。
- 真正绑定 MCP 协议的只有最外层壳：`internal/server/server.go` + `internal/tools/*.go`(把函数注册为 MCP tool)。
- 底层 `internal/xyzclient/`(HTTP 调用 + 鉴权)与协议无关，可原样复用。

因此只需把「MCP server 壳」旁边再加一层「CLI 子命令壳」，配一个 `SKILL.md` 即可。

## 3. 方案选型

- **方案 A：Skill 完全替代 MCP** — 改动大，丢失跨客户端可移植性。
- **方案 B(采用)：保留 MCP，新增 CLI 兼容层** — MCP server 与 CLI 共存，共用同一套 client 与鉴权。
  Claude Desktop / Cursor 继续用 MCP，Claude Code 用 Skill。改动小、零能力损失。

## 4. 架构

```
cmd/xiaoyuzhoufm-mcp/main.go     入口路由: init / cli / (默认)server
                    ├─ init      交互式登录, 写 token.json      [原有]
                    ├─ cli       CLI 兼容层, 结果打 JSON 到 stdout [新增]
                    └─ (default) MCP stdio server               [原有]
cmd/xiaoyuzhoufm-mcp/cli.go      CLI 子命令实现 (新增)
internal/xyzclient/              HTTP + 鉴权 + API (三种模式共用, 未改动)
internal/tools/ internal/server/ MCP 工具注册层 (未改动)
skills/xiaoyuzhoufm/SKILL.md     Skill 说明书 (新增)
```

三种模式复用同一 `xyzclient` 与同一 `token.json`，鉴权逻辑零重复。

## 5. 工具 → CLI 命令映射

| MCP 工具 | CLI 子命令 | 底层 client 函数 |
|---|---|---|
| `search_podcasts` | `cli search-podcast --keyword` | `SearchPodcasts` |
| `search_episodes` | `cli search-episode --keyword [--pid]` | `SearchEpisodes` |
| `search_users` | `cli search-user --keyword` | `SearchUsers` |
| `get_podcast_details` | `cli podcast-detail --pid` | `GetPodcastDetailsByID` |
| `get_episode_details` | `cli episode-detail --eid` | `GetEpisodeDetailsByID` |
| `list_podcast_episodes` | `cli list-episodes --pid [--order]` | `ListPodcastEpisodes` |
| `get_user_profile_by_id` | `cli user-profile --uid` | `GetUserProfileByID` |
| `get_user_stats` | `cli user-stats --uid` | `GetUserStats` |

每个 CLI 子命令 = 对应 MCP handler 的薄封装：把 stdin 参数换成 flag，把 MCP 结果换成打印 JSON。

## 6. 实现步骤

1. **新增 `cmd/xiaoyuzhoufm-mcp/cli.go`**
   - `runCLI(args)`：子命令路由 + 先加载 token(复用 server 模式逻辑)。
   - 每个子命令用独立 `flag.FlagSet` 解析参数、做必填校验、调 `xyzclient`、`emit` 打印 JSON。
   - `--brief`：把结果转 map 后对 `data` 列表按字段白名单裁剪(见下)。
2. **`main.go` 加 `cli` 分支**：`os.Args[1] == "cli"` 时进 `runCLI(os.Args[2:])`。
3. **编译**：`go build -o bin/xiaoyuzhoufm-mcp ./cmd/xiaoyuzhoufm-mcp`
4. **登录一次**：`./bin/xiaoyuzhoufm-mcp init`
5. **建 Skill**：`skills/xiaoyuzhoufm/SKILL.md`(含 frontmatter、二进制路径、命令表、用法约定)。
   放个人级 `~/.claude/skills/` 或项目级 `.claude/skills/` 生效。
6. **验证**：逐命令跑通 + 确认 MCP server 模式无回归 + `go vet`。

## 7. 关键设计点

- **鉴权零改动**：CLI 模式 `loadTokenForCLI()` 直接复用 `GetTokenManager` + `LoadTokenFromPath`，
  与 server 模式路径一致，token 刷新能力自动继承。
- **`--brief` 省 token**：搜索结果原始体积常达数十 KB。`briefify()` 只保留
  `title/pid/eid/uid/author/nickname/brief/subscriptionCount/episodeCount/duration/pubDate` 等关键字段，
  从根本上避免撑爆上下文。这是 Skill(走 Bash 文本) 相比 MCP 更需要关注的点。
- **参数校验补偿**：MCP 有 JSON Schema 强校验，CLI 靠模型拼命令行、易错。
  故每个子命令都做必填校验并给中文报错，`SKILL.md` 里逐命令列清参数与示例。

## 8. 注意事项 / 已知取舍

1. **参数校验弱于 MCP**：靠 CLI 内校验 + SKILL.md 文档缓解。
2. **输出体积**：默认引导用 `--brief`，需要时管道接 `jq` 进一步裁剪。
3. **可移植性**：Skill 仅在支持它的环境可用；保留 MCP 即为覆盖其他客户端。
4. **二进制路径硬编码**：`SKILL.md` 内写死了绝对路径，迁移机器需同步更新。

## 9. 验证记录

- `go build` / `go vet ./cmd/...` 通过。
- `search-podcast/episode/user`、`podcast-detail`、`list-episodes`、`user-*` 均返回正常 JSON。
- `--brief` 生效，字段显著精简。
- 缺 `--keyword` 等必填项时给出明确中文报错。
- 空 stdin 下 MCP server 模式返回标准 JSON-RPC Parse error(正常行为)，无崩溃回归。
