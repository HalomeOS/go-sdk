package go_sdk

import (
	"testing"
	"time"
)

func TestDownloadFileRange(t *testing.T) {
	// 示例配置 - 包含AuthToken
	config := DownloadConfigRange{
		URL:        "https://test.wukongyun.fun/v1/cat?id=QmXYCU7D1t7uQjFErBi3zPwPYqhdhU7GG2cJE4nuFt8unG", // 替换为实际的大文件URL
		OutputPath: "./testfile.zip",                                                                      // 本地保存路径
		ChunkSize:  10 * 1024 * 1024,                                                                      // 5MB分片
		Timeout:    60 * time.Second,                                                                      // 超时时间 		// 重试次数
		AuthToken:  "1b1427d1c3a88c308a0e0b3d61cf337e",                                                    // 替换为实际的授权令牌
	}

	// 执行下载
	if err := DownloadFileRange(config); err != nil {
		t.Logf("下载失败: %v\n", err)
		return
	}
	t.Logf("下载成功: %s\n", config.OutputPath)
}
