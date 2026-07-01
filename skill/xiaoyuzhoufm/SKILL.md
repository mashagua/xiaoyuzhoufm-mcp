---
name: xiaoyuzhoufm
description: 查询小宇宙播客数据。当用户想搜索播客/单集/用户、查订阅数、
  看播客或单集详情、对比多个播客、做播客排行榜时使用。
---

# 小宇宙 FM 查询

通过预编译的 CLI 调用小宇宙(xiaoyuzhoufm)接口查询播客数据。
鉴权已通过本地 token 自动处理，无需在命令里传任何凭证。

## 二进制

命令名：`xiaoyuzhoufm-mcp`（已安装到 PATH，可直接调用，无需路径前缀）。
所有查询命令形如 `xiaoyuzhoufm-mcp cli <子命令> [参数]`。

若提示 command not found，说明未安装到 PATH，在项目根目录执行：

```
go build -o ~/.local/bin/xiaoyuzhoufm-mcp ./cmd/xiaoyuzhoufm-mcp
```

（`~/.local/bin` 需在 PATH 中；换机器同样跑这一条即可，本文档无需改动。）

## 前置条件

首次使用需登录一次（交互式，输入手机号+验证码）：

```
xiaoyuzhoufm-mcp init
```

token 存于 `~/.mcp/xiaoyuzhoufm-mcp/token.json`，会自动刷新，之后无需重复登录。
若查询报「未找到 token」或「not authenticated」，提示用户先跑 `xiaoyuzhoufm-mcp init`。

## 命令一览

| 子命令 | 必填参数 | 可选参数 | 说明 |
|---|---|---|---|
| `search-podcast` | `--keyword <词>` | `--brief` | 按关键词搜播客 |
| `search-episode` | `--keyword <词>` | `--pid <PID>` `--brief` | 搜单集，可限定某播客内 |
| `search-user` | `--keyword <词>` | `--brief` | 搜用户 |
| `podcast-detail` | `--pid <PID>` | — | 播客详情 |
| `episode-detail` | `--eid <EID>` | — | 单集详情 |
| `list-episodes` | `--pid <PID>` | `--order asc\|desc` `--brief` | 播客的单集列表 |
| `user-profile` | `--uid <UID>` | — | 用户公开资料 |
| `user-stats` | `--uid <UID>` | — | 用户统计(关注/粉丝/订阅数等) |

分页(可选)：搜索类命令支持 `--search-id <ID>` 和 `--load-more-key <数字>`，
值来自上一次搜索结果里的 `loadMoreKey` 字段。

## 输出

结果为 JSON 打到 stdout。关键字段：

- 播客：`pid`(唯一ID) `title` `author` `subscriptionCount`(订阅数) `episodeCount`
- 单集：`eid`(唯一ID) `title` `duration`(秒) `pubDate`
- 用户：`uid`(唯一ID) `nickname`

## 重要用法约定

- **默认加 `--brief`**：搜索/列表结果原始体积可达数十 KB，`--brief` 只保留上述关键字段，
  避免撑爆上下文。只有当用户明确要完整字段(如简介全文、图片URL)时才省略。
- **PID/EID/UID 从哪来**：先 search 拿到 ID，再用 ID 查 detail。别凭空编造 ID。
- **排行榜/对比类需求**：多次 `search-podcast --brief`，收集各 `subscriptionCount` 后自行排序。
  注意同一团队常有衍生节目(如「XX｜番外」)，去重时按需保留主节目。
- **进一步裁剪**：需要提取特定字段时管道接 `jq`，例如
  `xiaoyuzhoufm-mcp cli search-podcast --keyword "商业" --brief | jq '.data[] | {title, subscriptionCount}'`

## 示例

查订阅数：
```
xiaoyuzhoufm-mcp cli search-podcast --keyword "纵横四海" --brief
```

看某播客最新几集：
```
xiaoyuzhoufm-mcp cli list-episodes --pid "62694abdb221dd5908417d1e" --order desc --brief
```

查用户统计：
```
xiaoyuzhoufm-mcp cli user-stats --uid "5eee1c0592d70237678dbd91"
```
