package go_sdk

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// 下载配置
type DownloadConfig struct {
	URL        string        // 下载URL
	Timeout    time.Duration // 超时时间
	RetryCount int           // 重试次数
	AuthToken  string        // 授权令牌
}

// 下载结果，包含数据流和文件信息
type DownloadResult struct {
	Reader    io.Reader    // 下载的数据流
	TotalSize int64        // 文件总大小（可能为0表示未知）
	Close     func() error // 关闭资源的函数
}

// 下载文件并返回数据流
func DownloadAsStream(ctx context.Context, config DownloadConfig) (*DownloadResult, error) {
	// 检查参数
	if config.URL == "" {
		return nil, fmt.Errorf("URL不能为空")
	}
	if config.Timeout <= 0 {
		config.Timeout = 300 * time.Second // 默认5分钟超时
	}
	if config.RetryCount < 0 {
		config.RetryCount = 3 // 默认重试3次
	}

	// 带重试的下载
	var resp *http.Response
	for attempt := 0; attempt <= config.RetryCount; attempt++ {
		if attempt > 0 {
			fmt.Printf("重试下载（第%d次重试）\n", attempt)
			time.Sleep(time.Duration(attempt) * time.Second) // 指数退避
		}

		// 创建请求
		req, err := http.NewRequestWithContext(ctx, "GET", config.URL, nil)
		if err != nil {
			return nil, fmt.Errorf("创建请求失败: %v", err)
		}

		// 添加AuthToken到请求头
		if config.AuthToken != "" {
			req.Header.Set("AuthToken", config.AuthToken)
		}

		// 发送请求
		client := &http.Client{}
		resp, err = client.Do(req)
		if err != nil {
			continue // 重试
		}

		// 检查响应状态
		if resp.StatusCode == http.StatusOK {
			break // 成功获取响应
		}

		// 非成功状态，关闭响应并继续重试
		resp.Body.Close()
		if attempt == config.RetryCount {
			return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
		}
	}

	if resp == nil {
		return nil, fmt.Errorf("无法建立有效的HTTP连接")
	}
	//获取文件大小
	sizeStr := resp.Header.Get("Content-Length")
	if sizeStr == "" {
		sizeStr = "0"
	}
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return nil, err
	}
	// 构建返回结果
	result := &DownloadResult{
		Reader:    resp.Body,
		TotalSize: size,
		Close: func() error {
			return resp.Body.Close()
		},
	}

	return result, nil
}

// 示例使用：读取返回的流并处理
func main() {
	// 配置
	config := DownloadConfig{
		URL:        "https://example.com/file.txt", // 替换为实际的文件URL
		Timeout:    60 * time.Second,               // 超时时间
		RetryCount: 3,                              // 重试次数
		AuthToken:  "your-auth-token-here",         // 替换为实际的授权令牌
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	// 下载并获取流
	resultBody, err := DownloadAsStream(ctx, config)
	if err != nil {
		fmt.Printf("下载失败: %v\n", err)
		return
	}
	defer resultBody.Close() // 确保资源最终会被关闭

	//do something
}
