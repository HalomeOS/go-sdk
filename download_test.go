package go_sdk

import (
	"context"
	"testing"
	"time"
)

func TestDownloadAsStream(t *testing.T) {
	// 配置
	config := DownloadConfig{
		URL:       "https://test.wukongyun.fun/v1/cat?id=QmNop3WXGspWUSbvC26xpDL6xh8nQjxLjX4vLHScdb3bWk", // 替换为实际的文件URL
		Timeout:   60 * time.Second,                                                                      // 超时时间 		// 重试次数
		AuthToken: "1b1427d1c3a88c308a0e0b3d61cf337e",                                                    // 替换为实际的授权令牌
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	// 下载并获取流
	resultBody, err := DownloadAsStream(ctx, config)
	if err != nil {
		t.Errorf("下载失败: %v\n", err)
		return
	}
	t.Logf("下载成功，文件大小: %d", resultBody.TotalSize)
	defer resultBody.Close() // 确保资源最终会被关闭
}
