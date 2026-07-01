package xyzclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"xiaoyuzhoufm-mcp/internal/constants" // For APIBaseURL
)

// RequestVerificationCode sends a request to Xiaoyuzhou API to send a verification code.
//
// 该接口走创作者后台（Web 端）域名 podcaster-api.xiaoyuzhoufm.com，使用浏览器请求头。
// App 域名 api.xiaoyuzhoufm.com 对无 token 的登录/发码接口做 App 版本强校验，
// 旧构建号会被拒绝并返回 code 1003「请升级到最新版本后重试登录」；
// Web 域名不做该校验，可稳定发码。换回的 token 仍可用于 App 数据接口。
func RequestVerificationCode(areaCode, phoneNumber string) error {
	apiURL := constants.PodcasterAPIBaseURL + "/v1/auth/send-code"
	slog.Debug("Requesting verification code")

	requestBody := sendCodeRequestBody{
		MobilePhoneNumber: phoneNumber,
		AreaCode:          areaCode,
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set Headers (创作者后台 Web 接口：浏览器头，不带 App 版本号)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6")
	req.Header.Set("Origin", constants.PodcasterOrigin)
	req.Header.Set("Referer", constants.PodcasterOrigin+"/")
	req.Header.Set("User-Agent", constants.WebUserAgent)

	slog.Debug("Sending HTTP request to sendCode API", "headers", req.Header, "body", string(jsonBody))
	resp, err := GetHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body from sendCode: %w", err)
	}
	slog.Debug("Received response from sendCode API", "statusCode", resp.StatusCode, "body", string(responseBodyBytes))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("api request failed with status %d: %s", resp.StatusCode, string(responseBodyBytes))
	}

	// For sendCode, if HTTP status is OK, assume success.
	// If the API *can* return 200 OK but still indicate a business error in the body
	// (e.g. {"error_no": 123, "error_message": "..."}), then we would need to parse responseBodyBytes
	// into a minimal error struct here and check. Based on current understanding, this is not needed for sendCode.
	slog.Debug("Verification code request successful (HTTP 200 OK).")
	return nil
}

// LoginWithCode sends the area code, phone number, and verification code to Xiaoyuzhou API to log in.
// It returns the access token, refresh token, UID, and nickname upon success.
//
// 与发码一致，走创作者后台（Web 端）域名 podcaster-api.xiaoyuzhoufm.com 绕开 App 版本校验。
// 登录成功后 token 仍在响应头 x-jike-access-token / x-jike-refresh-token 中返回，
// 这些 token 可继续用于 App 数据接口 api.xiaoyuzhoufm.com。
func LoginWithCode(areaCode, phoneNumber, code string) (accessToken, refreshToken, uid, nickname string, err error) {
	apiURL := constants.PodcasterAPIBaseURL + "/v1/auth/login-with-sms"
	slog.Debug("Attempting to login with code")

	requestBody := loginOrSignUpWithSMSRequestBody{
		AreaCode:          areaCode,
		VerifyCode:        code,
		MobilePhoneNumber: phoneNumber,
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set Headers (创作者后台 Web 接口：浏览器头，不带 App 版本号)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6")
	req.Header.Set("Origin", constants.PodcasterOrigin)
	req.Header.Set("Referer", constants.PodcasterOrigin+"/")
	req.Header.Set("User-Agent", constants.WebUserAgent)

	slog.Debug("Sending HTTP request to login API", "headers", req.Header, "body", string(jsonBody))
	resp, err := GetHTTPClient().Do(req)
	if err != nil {
		return "", "", "", "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// IMPORTANT: Tokens are in headers for login, user info in body.
	accessToken = resp.Header.Get("x-jike-access-token")
	refreshToken = resp.Header.Get("x-jike-refresh-token")

	responseBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to read response body: %w", err)
	}
	slog.Debug("Received response from login API", "statusCode", resp.StatusCode, "headers", resp.Header, "body", string(responseBodyBytes))

	if resp.StatusCode != http.StatusOK {
		return "", "", "", "", fmt.Errorf("api request failed with status %d: %s", resp.StatusCode, string(responseBodyBytes))
	}

	// If HTTP status is OK, parse the body for user info.
	var apiRespBody LoginAPIResponse
	if err := json.Unmarshal(responseBodyBytes, &apiRespBody); err != nil {
		return "", "", "", "", fmt.Errorf("failed to unmarshal success response JSON body: %w", err)
	}

	uid = apiRespBody.Data.User.UID
	nickname = apiRespBody.Data.User.Nickname

	if accessToken == "" || refreshToken == "" {
		return "", "", "", "", fmt.Errorf("login successful but tokens are empty in headers")
	}
	if uid == "" {
		return "", "", "", "", fmt.Errorf("login successful but UID is empty in body")
	}

	slog.Debug("Login successful.", "uid", uid, "nickname", nickname)
	return accessToken, refreshToken, uid, nickname, nil
}

// PerformTokenRefresh sends the refresh token to Xiaoyuzhou API to get a new access token.
// It returns the new access token and a new refresh token.
func PerformTokenRefresh(currentRefreshToken string) (newAccessToken, newRefreshToken string, err error) {
	apiURL := constants.APIBaseURL + "/app_auth_tokens.refresh"
	slog.Debug("Attempting to refresh token")

	req, err := http.NewRequest(http.MethodPost, apiURL, nil) // No body
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set Headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	req.Header.Set("Host", "api.xiaoyuzhoufm.com")
	req.Header.Set("User-Agent", constants.XiaoyuzhouUserAgent)
	req.Header.Set("x-jike-refresh-token", currentRefreshToken)
	req.Header.Set("Market", "AppStore")
	req.Header.Set("App-BuildNo", constants.XiaoyuzhouAppBuildNo)
	req.Header.Set("OS", "ios")
	req.Header.Set("Manufacturer", "Apple")
	req.Header.Set("BundleID", "app.podcast.cosmos")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Accept-Language", "zh-Hant-HK;q=1.0, zh-Hans-CN;q=0.9")
	req.Header.Set("Model", "iPhone14,2")
	req.Header.Set("app-permissions", "4")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("App-Version", constants.XiaoyuzhouAppVersion)
	req.Header.Set("WifiConnected", "true")
	req.Header.Set("OS-Version", "17.4.1")
	req.Header.Set("x-custom-xiaoyuzhou-app-dev", "")

	slog.Debug("Sending HTTP request to token refresh API", "headers", req.Header)
	resp, err := GetHTTPClient().Do(req)
	if err != nil {
		return "", "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %w", err)
	}
	slog.Debug("Received response from token refresh API", "statusCode", resp.StatusCode, "body", string(responseBodyBytes))

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("api request failed with status %d: %s", resp.StatusCode, string(responseBodyBytes))
	}

	var apiResp RefreshTokenAPIResponse
	if err := json.Unmarshal(responseBodyBytes, &apiResp); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal success response JSON: %w", err)
	}

	if !apiResp.Success {
		return "", "", fmt.Errorf("api reported token refresh not successful (success: false)")
	}

	newAccessToken = apiResp.XJikeAccessToken
	newRefreshToken = apiResp.XJikeRefreshToken

	if newAccessToken == "" || newRefreshToken == "" {
		return "", "", fmt.Errorf("token refresh successful but new tokens are empty in response")
	}

	slog.Debug("Token refresh successful.")
	return newAccessToken, newRefreshToken, nil
}
