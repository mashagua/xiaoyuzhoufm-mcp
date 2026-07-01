package constants

import "os"

const APIBaseURL = "https://api.xiaoyuzhoufm.com"

// PodcasterAPIBaseURL 是创作者后台（Web 端）接口域名。
// 登录/发码等无 token 接口走这里：该域名不做 App 版本强校验，
// 可绕开 App 接口因版本过旧返回的 code 1003「请升级到最新版本后重试登录」。
const PodcasterAPIBaseURL = "https://podcaster-api.xiaoyuzhoufm.com"

// PodcasterOrigin 是创作者后台 Web 页面来源，作为登录/发码请求的 Origin/Referer。
const PodcasterOrigin = "https://podcaster.xiaoyuzhoufm.com"

// WebUserAgent 是登录/发码 Web 接口使用的浏览器 UA（不含 App 版本号）。
const WebUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36 Edg/146.0.0.0"

const FixedDeviceID = "81ADBFD6-6921-482B-9AB9-A29E7CC7BB55"

var XiaoyuzhouAppVersion = envOrDefault("XIAOYUZHOU_APP_VERSION", "2.111.0")
var XiaoyuzhouAppBuildNo = envOrDefault("XIAOYUZHOU_APP_BUILD_NO", "211100")
var XiaoyuzhouUserAgent = "Xiaoyuzhou/" + XiaoyuzhouAppVersion + " (build:" + XiaoyuzhouAppBuildNo + "; iOS 17.4.1)"

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
