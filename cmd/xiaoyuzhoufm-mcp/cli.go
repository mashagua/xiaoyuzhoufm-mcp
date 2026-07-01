package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"

	"xiaoyuzhoufm-mcp/internal/xyzclient"
)

// runCLI 是 CLI 兼容层的入口，供 Skill / 命令行直接调用。
// 与 MCP server 模式共用同一套鉴权（token.json）和 xyzclient，
// 区别仅在于：结果以 JSON 打印到 stdout，参数通过命令行 flag 传入。
func runCLI(args []string) {
	if len(args) < 1 {
		printCLIUsage()
		os.Exit(2)
	}

	// 与 server 模式一致：先加载 token（含自动刷新能力）。
	if err := loadTokenForCLI(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr, "请先运行 './xiaoyuzhoufm-mcp init' 登录。")
		os.Exit(1)
	}

	subcommand := args[0]
	rest := args[1:]

	switch subcommand {
	case "search-podcast":
		cliSearchPodcast(rest)
	case "search-episode":
		cliSearchEpisode(rest)
	case "search-user":
		cliSearchUser(rest)
	case "podcast-detail":
		cliPodcastDetail(rest)
	case "episode-detail":
		cliEpisodeDetail(rest)
	case "list-episodes":
		cliListEpisodes(rest)
	case "user-profile":
		cliUserProfile(rest)
	case "user-stats":
		cliUserStats(rest)
	default:
		fmt.Fprintf(os.Stderr, "未知子命令: %s\n", subcommand)
		printCLIUsage()
		os.Exit(2)
	}
}

// loadTokenForCLI 复用 server 模式的 token 加载逻辑。
func loadTokenForCLI() error {
	tm, err := xyzclient.GetTokenManager()
	if err != nil {
		return fmt.Errorf("初始化 token manager 失败: %w", err)
	}
	userTokenPath, err := xyzclient.GetUserTokenPath()
	if err != nil {
		return fmt.Errorf("无法确定 token 路径: %w", err)
	}
	if err := tm.LoadTokenFromPath(userTokenPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("未找到 token 文件 (%s)", userTokenPath)
		}
		return fmt.Errorf("加载 token 失败 (%s): %w", userTokenPath, err)
	}
	slog.Debug("CLI 模式 token 加载成功", "path", userTokenPath)
	return nil
}

func printCLIUsage() {
	fmt.Fprintln(os.Stderr, `用法: xiaoyuzhoufm-mcp cli <子命令> [参数]

子命令:
  search-podcast   --keyword <关键词> [--brief]
  search-episode   --keyword <关键词> [--pid <PID>] [--brief]
  search-user      --keyword <关键词> [--brief]
  podcast-detail   --pid <PID>
  episode-detail   --eid <EID>
  list-episodes    --pid <PID> [--order asc|desc] [--brief]
  user-profile     --uid <UID>
  user-stats       --uid <UID>

通用:
  --brief          仅保留关键字段(title/pid/eid/uid/subscriptionCount 等)，省 token`)
}

// --- 输出辅助 ---

// emit 将结果序列化为 JSON 打印到 stdout。brief=true 时先裁剪字段。
func emit(v interface{}, brief bool) {
	if brief {
		v = briefify(v)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "序列化结果失败: %v\n", err)
		os.Exit(1)
	}
}

// fail 打印错误并退出。
func fail(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	} else {
		fmt.Fprintln(os.Stderr, msg)
	}
	os.Exit(1)
}

// briefify 把结果转成 map 后，对 data 列表只保留白名单字段，
// 大幅缩小输出体积（搜索结果常达数十 KB）。
func briefify(v interface{}) interface{} {
	raw, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return v
	}
	keep := map[string]bool{
		"title": true, "pid": true, "eid": true, "uid": true,
		"author": true, "nickname": true, "brief": true,
		"subscriptionCount": true, "episodeCount": true,
		"duration": true, "pubDate": true, "latestEpisodePubDate": true,
		"loadMoreKey": true,
	}
	if data, ok := m["data"].([]interface{}); ok {
		trimmed := make([]interface{}, 0, len(data))
		for _, item := range data {
			if obj, ok := item.(map[string]interface{}); ok {
				small := map[string]interface{}{}
				for k, val := range obj {
					if keep[k] {
						small[k] = val
					}
				}
				trimmed = append(trimmed, small)
			} else {
				trimmed = append(trimmed, item)
			}
		}
		out := map[string]interface{}{"data": trimmed}
		if lmk, ok := m["loadMoreKey"]; ok {
			out["loadMoreKey"] = lmk
		}
		return out
	}
	return m
}

// searchLoadMoreKey 从 flag 构造搜索分页键(可选)。
func searchLoadMoreKey(searchID string, loadMoreKey int) *xyzclient.SearchAPILoadMoreKey {
	if searchID == "" && loadMoreKey == 0 {
		return nil
	}
	return &xyzclient.SearchAPILoadMoreKey{
		LoadMoreKey: loadMoreKey,
		SearchID:    searchID,
	}
}

// --- 子命令实现 ---

func cliSearchPodcast(args []string) {
	fs := flag.NewFlagSet("search-podcast", flag.ExitOnError)
	keyword := fs.String("keyword", "", "搜索关键词(必填)")
	searchID := fs.String("search-id", "", "分页会话ID(可选)")
	loadMoreKey := fs.Int("load-more-key", 0, "分页键(可选)")
	brief := fs.Bool("brief", false, "仅保留关键字段")
	_ = fs.Parse(args)

	if *keyword == "" {
		fail("参数 --keyword 不能为空", nil)
	}
	res, err := xyzclient.SearchPodcasts(*keyword, searchLoadMoreKey(*searchID, *loadMoreKey))
	if err != nil {
		fail("搜索播客失败", err)
	}
	emit(res, *brief)
}

func cliSearchEpisode(args []string) {
	fs := flag.NewFlagSet("search-episode", flag.ExitOnError)
	keyword := fs.String("keyword", "", "搜索关键词(必填)")
	pid := fs.String("pid", "", "限定在某播客内搜索(可选)")
	searchID := fs.String("search-id", "", "分页会话ID(可选)")
	loadMoreKey := fs.Int("load-more-key", 0, "分页键(可选)")
	brief := fs.Bool("brief", false, "仅保留关键字段")
	_ = fs.Parse(args)

	if *keyword == "" {
		fail("参数 --keyword 不能为空", nil)
	}
	res, err := xyzclient.SearchEpisodes(*keyword, *pid, searchLoadMoreKey(*searchID, *loadMoreKey))
	if err != nil {
		fail("搜索单集失败", err)
	}
	emit(res, *brief)
}

func cliSearchUser(args []string) {
	fs := flag.NewFlagSet("search-user", flag.ExitOnError)
	keyword := fs.String("keyword", "", "搜索关键词(必填)")
	searchID := fs.String("search-id", "", "分页会话ID(可选)")
	loadMoreKey := fs.Int("load-more-key", 0, "分页键(可选)")
	brief := fs.Bool("brief", false, "仅保留关键字段")
	_ = fs.Parse(args)

	if *keyword == "" {
		fail("参数 --keyword 不能为空", nil)
	}
	res, err := xyzclient.SearchUsers(*keyword, searchLoadMoreKey(*searchID, *loadMoreKey))
	if err != nil {
		fail("搜索用户失败", err)
	}
	emit(res, *brief)
}

func cliPodcastDetail(args []string) {
	fs := flag.NewFlagSet("podcast-detail", flag.ExitOnError)
	pid := fs.String("pid", "", "播客ID(必填)")
	_ = fs.Parse(args)

	if *pid == "" {
		fail("参数 --pid 不能为空", nil)
	}
	res, err := xyzclient.GetPodcastDetailsByID(*pid)
	if err != nil {
		fail("获取播客详情失败", err)
	}
	emit(res, false)
}

func cliEpisodeDetail(args []string) {
	fs := flag.NewFlagSet("episode-detail", flag.ExitOnError)
	eid := fs.String("eid", "", "单集ID(必填)")
	_ = fs.Parse(args)

	if *eid == "" {
		fail("参数 --eid 不能为空", nil)
	}
	res, err := xyzclient.GetEpisodeDetailsByID(*eid)
	if err != nil {
		fail("获取单集详情失败", err)
	}
	emit(res, false)
}

func cliListEpisodes(args []string) {
	fs := flag.NewFlagSet("list-episodes", flag.ExitOnError)
	pid := fs.String("pid", "", "播客ID(必填)")
	order := fs.String("order", "desc", "排序 asc|desc")
	brief := fs.Bool("brief", false, "仅保留关键字段")
	_ = fs.Parse(args)

	if *pid == "" {
		fail("参数 --pid 不能为空", nil)
	}
	if *order != "asc" && *order != "desc" {
		fail("参数 --order 必须是 asc 或 desc", nil)
	}
	res, err := xyzclient.ListPodcastEpisodes(xyzclient.EpisodeListRequest{
		PID:   *pid,
		Order: *order,
		Limit: 20,
	})
	if err != nil {
		fail("获取单集列表失败", err)
	}
	emit(res, *brief)
}

func cliUserProfile(args []string) {
	fs := flag.NewFlagSet("user-profile", flag.ExitOnError)
	uid := fs.String("uid", "", "用户ID(必填)")
	_ = fs.Parse(args)

	if *uid == "" {
		fail("参数 --uid 不能为空", nil)
	}
	res, err := xyzclient.GetUserProfileByID(*uid)
	if err != nil {
		fail("获取用户资料失败", err)
	}
	emit(res, false)
}

func cliUserStats(args []string) {
	fs := flag.NewFlagSet("user-stats", flag.ExitOnError)
	uid := fs.String("uid", "", "用户ID(必填)")
	_ = fs.Parse(args)

	if *uid == "" {
		fail("参数 --uid 不能为空", nil)
	}
	res, err := xyzclient.GetUserStats(*uid)
	if err != nil {
		fail("获取用户统计失败", err)
	}
	emit(res, false)
}
