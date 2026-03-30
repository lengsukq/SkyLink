package easytier

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Release 表示 GitHub 上一个 release 的简要信息
type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// FetchLatestRelease 请求 GitHub API 获取最新 release tag
func FetchLatestRelease() (tagName, htmlURL string, err error) {
	req, err := http.NewRequest(http.MethodGet, GitHubReleasesAPI, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("github api: %s", resp.Status)
	}
	var v struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", "", err
	}
	return v.TagName, v.HTMLURL, nil
}

// FetchReleases 请求 GitHub API 获取 releases 列表，用于版本下拉；perPage 建议 30
func FetchReleases(perPage int) ([]Release, error) {
	if perPage <= 0 {
		perPage = 30
	}
	url := fmt.Sprintf("%s?per_page=%d", GitHubReleasesListAPI, perPage)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github releases api: %s", resp.Status)
	}
	var list []Release
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}
	return list, nil
}

