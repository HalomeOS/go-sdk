package go_sdk

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

// DownloadConfigRange 下载配置
type DownloadConfigRange struct {
	URL        string        // 下载URL
	OutputPath string        // 输出文件路径
	ChunkSize  int64         // 分片大小
	Timeout    time.Duration // 超时时间
	AuthToken  string        // 授权令牌
}

// DownloadFileRange 简化的串行分片下载文件函数
func DownloadFileRange(config DownloadConfigRange) error {
	// 参数校验与默认值设置
	if config.URL == "" || config.OutputPath == "" {
		return fmt.Errorf("URL和输出路径不能为空")
	}
	if config.ChunkSize <= 0 {
		config.ChunkSize = 5 * 1024 * 1024 // 默认5MB
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}

	// 获取文件总大小
	fileSize, err := getFileSize(config.URL, config.AuthToken)
	if err != nil {
		return fmt.Errorf("获取文件大小失败: %v", err)
	}
	fmt.Printf("文件总大小: %.2f MB\n", float64(fileSize)/1024/1024)

	// 检查断点续传
	downloadedSize, err := getFileSizeLocal(config.OutputPath)
	if err != nil {
		return fmt.Errorf("检查已下载大小失败: %v", err)
	}
	if downloadedSize == fileSize {
		fmt.Println("文件已完全下载")
		return nil
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	// 下载所有分片
	totalChunks := (fileSize + config.ChunkSize - 1) / config.ChunkSize
	startChunk := downloadedSize / config.ChunkSize

	fmt.Printf("分片大小: %.2f MB, 总分片数: %d, 已完成: %d, 剩余: %d\n",
		float64(config.ChunkSize)/1024/1024, totalChunks, startChunk, totalChunks-startChunk)
	// 添加累计下载字节数变量
	totalDownloaded := downloadedSize
	for i := startChunk; i < totalChunks; i++ {
		if ctx.Err() != nil {
			return fmt.Errorf("下载被中断: %v", ctx.Err())
		}

		start := i * config.ChunkSize
		if i == startChunk {
			start = downloadedSize // 从上次中断处继续
		}
		end := start + config.ChunkSize - 1
		if end >= fileSize {
			end = fileSize - 1
		}

		// 单次下载尝试
		bytesWritten, err := downloadChunk(ctx, config.URL, config.OutputPath, start, end, config.AuthToken)
		if err != nil {
			return fmt.Errorf("分片 %d (范围: %d-%d) 下载失败: %v", i+1, start, end, err)
		}

		// 更新累计下载字节数
		totalDownloaded += bytesWritten

		// 正确的进度计算
		progress := float64(totalDownloaded) / float64(fileSize) * 100
		fmt.Printf("\r下载进度: %.2f%%", progress)
	}

	// 验证最终文件大小
	finalSize, err := getFileSizeLocal(config.OutputPath)
	if err != nil {
		return fmt.Errorf("验证文件大小失败: %v", err)
	}
	if finalSize != fileSize {
		return fmt.Errorf("下载不完整，期望大小: %d, 实际大小: %d", fileSize, finalSize)
	}
	fmt.Println("\n文件下载完成!")
	return nil
}

// downloadChunk 下载单个分片的核心方法
func downloadChunk(ctx context.Context, url, outputPath string, start, end int64, authToken string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	if authToken != "" {
		req.Header.Set("AuthToken", authToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查服务器响应
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	// 打开文件并写入数据
	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return 0, fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return 0, fmt.Errorf("移动文件指针失败: %v", err)
	}

	writer := bufio.NewWriterSize(file, 1024*1024)
	bytesWritten, err := io.Copy(writer, resp.Body)
	if err != nil {
		return bytesWritten, fmt.Errorf("写入文件失败: %v", err)
	}

	if err := writer.Flush(); err != nil {
		fmt.Printf("警告: 刷新缓冲区时出错: %v\n", err)
	}

	return bytesWritten, nil
}

// getFileSize 获取远程文件大小
func getFileSize(url, authToken string) (int64, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return 0, err
	}
	if authToken != "" {
		req.Header.Set("AuthToken", authToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("获取文件信息失败，状态码: %d", resp.StatusCode)
	}

	sizeStr := resp.Header.Get("Content-Length")
	if sizeStr == "" {
		return 0, fmt.Errorf("服务器未提供Content-Length")
	}
	return strconv.ParseInt(sizeStr, 10, 64)
}

// getFileSizeLocal 获取本地文件大小
func getFileSizeLocal(outputPath string) (int64, error) {
	fileInfo, err := os.Stat(outputPath)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("获取文件信息失败: %v", err)
	}
	return fileInfo.Size(), nil
}
