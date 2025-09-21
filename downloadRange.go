package go_sdk

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// 下载配置 - 新增AuthToken字段
type DownloadConfigRange struct {
	URL        string        // 下载URL
	OutputPath string        // 输出文件路径
	ChunkSize  int64         // 分片大小
	Timeout    time.Duration // 超时时间
	RetryCount int           // 重试次数
	AuthToken  string        // 授权令牌
}

// 检查服务器是否支持Range - 增加AuthToken头
func checkRangeSupport(url string, authToken string) (bool, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false, fmt.Errorf("创建HEAD请求失败: %v", err)
	}

	// 添加AuthToken到请求头
	if authToken != "" {
		req.Header.Set("AuthToken", authToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("发送HEAD请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查是否支持Range
	acceptRanges := resp.Header.Get("Accept-Ranges")
	return acceptRanges == "bytes", nil
}

// 获取已下载的大小（用于断点续传）
func getDownloadedSize(outputPath string) (int64, error) {
	fileInfo, err := os.Stat(outputPath)
	if os.IsNotExist(err) {
		return 0, nil // 文件不存在，已下载大小为0
	}
	if err != nil {
		return 0, fmt.Errorf("获取文件信息失败: %v", err)
	}
	return fileInfo.Size(), nil
}

// 下载指定范围，支持重试 - 增加AuthToken参数
func downloadChunk(ctx context.Context, url, outputPath string, start, end int64,
	totalSize int64, retryCount int, authToken string) (int64, error) {
	var err error
	bytesDownloaded := int64(0)

	// 带重试的下载
	for attempt := 0; attempt <= retryCount; attempt++ {
		if attempt > 0 {
			fmt.Printf("重试下载分片 %d-%d (第%d次重试)\n", start, end, attempt)
			time.Sleep(time.Duration(attempt) * time.Second) // 指数退避
		}

		bytesDownloaded, err = downloadRangeWithCtx(ctx, url, outputPath, start, end, authToken)
		if err == nil {
			// 显示进度
			progress := float64(start+bytesDownloaded) / float64(totalSize) * 100
			fmt.Printf("\r下载进度: %.2f%%", progress)
			return bytesDownloaded, nil
		}

		// 检查上下文是否已取消
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
	}

	return 0, fmt.Errorf("分片 %d-%d 下载失败，已达最大重试次数: %v", start, end, err)
}

// 带上下文的范围下载 - 增加AuthToken头
func downloadRangeWithCtx(ctx context.Context, url, outputPath string, start, end int64, authToken string) (int64, error) {
	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置Range头
	rangeHeader := fmt.Sprintf("bytes=%d-%d", start, end)
	req.Header.Set("Range", rangeHeader)

	// 添加AuthToken到请求头
	if authToken != "" {
		req.Header.Set("AuthToken", authToken)
	}

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusPartialContent {
		return 0, fmt.Errorf("请求失败，状态码: %d，期望: %d",
			resp.StatusCode, http.StatusPartialContent)
	}

	// 验证返回的内容范围是否正确
	contentRange := resp.Header.Get("Content-Range")
	if !validateContentRange(contentRange, start, end) {
		return 0, fmt.Errorf("返回的内容范围不匹配，期望: %d-%d，实际: %s",
			start, end, contentRange)
	}

	// 打开文件，准备写入
	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return 0, fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	// 移动到文件指定位置写入
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return 0, fmt.Errorf("移动文件指针失败: %v", err)
	}

	// 写入数据
	writer := bufio.NewWriterSize(file, 1024*1024) // 1MB缓冲区
	defer writer.Flush()

	bytesWritten, err := io.Copy(writer, resp.Body)
	if err != nil {
		return bytesWritten, fmt.Errorf("写入文件失败: %v", err)
	}

	// 验证写入的字节数
	expectedSize := end - start + 1
	if bytesWritten != expectedSize {
		return bytesWritten, fmt.Errorf("写入字节数不匹配，期望: %d, 实际: %d", expectedSize, bytesWritten)
	}

	return bytesWritten, nil
}

// 验证Content-Range是否与请求的范围一致
func validateContentRange(contentRange string, expectedStart, expectedEnd int64) bool {
	if contentRange == "" {
		return false
	}

	// Content-Range格式: bytes start-end/total
	parts := strings.Split(contentRange, " ")
	if len(parts) != 2 || parts[0] != "bytes" {
		return false
	}

	rangePart := strings.Split(parts[1], "/")[0]
	rangeParts := strings.Split(rangePart, "-")
	if len(rangeParts) != 2 {
		return false
	}

	start, err1 := strconv.ParseInt(rangeParts[0], 10, 64)
	end, err2 := strconv.ParseInt(rangeParts[1], 10, 64)

	return err1 == nil && err2 == nil && start == expectedStart && end == expectedEnd
}

// 获取文件总大小 - 增加AuthToken头
func getFileSize(url string, authToken string) (int64, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return 0, err
	}

	// 添加AuthToken到请求头
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

	// 从Content-Length头获取文件大小
	sizeStr := resp.Header.Get("Content-Length")
	if sizeStr == "" {
		return 0, fmt.Errorf("服务器未提供Content-Length")
	}

	return strconv.ParseInt(sizeStr, 10, 64)
}

// 串行分片下载文件
func DownloadFileRange(config DownloadConfigRange) error {
	// 检查参数
	if config.URL == "" || config.OutputPath == "" {
		return fmt.Errorf("URL和输出路径不能为空")
	}
	if config.ChunkSize <= 0 {
		config.ChunkSize = 5 * 1024 * 1024 // 默认5MB
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second // 默认30秒超时
	}
	if config.RetryCount < 0 {
		config.RetryCount = 3 // 默认重试3次
	}

	// 检查服务器是否支持Range - 传入AuthToken
	supportRange, err := checkRangeSupport(config.URL, config.AuthToken)
	if err != nil {
		return fmt.Errorf("检查Range支持失败: %v", err)
	}
	if !supportRange {
		return fmt.Errorf("服务器不支持Range请求，无法进行分片下载")
	}

	// 获取文件总大小 - 传入AuthToken
	fileSize, err := getFileSize(config.URL, config.AuthToken)
	if err != nil {
		return fmt.Errorf("获取文件大小失败: %v", err)
	}
	fmt.Printf("文件总大小: %.2f MB\n", float64(fileSize)/1024/1024)

	// 检查断点续传
	downloadedSize, err := getDownloadedSize(config.OutputPath)
	if err != nil {
		return fmt.Errorf("检查已下载大小失败: %v", err)
	}

	if downloadedSize > 0 {
		if downloadedSize == fileSize {
			fmt.Println("文件已完全下载，无需继续")
			return nil
		}
		fmt.Printf("检测到部分下载，已下载: %.2f MB，将继续下载剩余部分\n",
			float64(downloadedSize)/1024/1024)
	}

	// 计算需要下载的分片
	totalChunks := (fileSize + config.ChunkSize - 1) / config.ChunkSize
	startChunk := downloadedSize / config.ChunkSize

	fmt.Printf("分片大小: %.2f MB, 总分片数: %d, 已完成: %d, 剩余: %d\n",
		float64(config.ChunkSize)/1024/1024,
		totalChunks,
		startChunk,
		totalChunks-startChunk)

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	// 串行下载每个分片
	for i := startChunk; i < totalChunks; i++ {
		// 检查上下文是否已取消（如超时）
		if ctx.Err() != nil {
			return fmt.Errorf("下载被中断: %v", ctx.Err())
		}

		start := i * config.ChunkSize
		end := start + config.ChunkSize - 1
		if end >= fileSize {
			end = fileSize - 1
		}

		// 如果该分片已下载，则跳过
		if start < downloadedSize {
			// 检查整个分片是否已下载
			if end < downloadedSize {
				continue
			}
			// 部分下载，从已下载位置开始
			start = downloadedSize
		}

		// 下载当前分片 - 传入AuthToken
		fmt.Printf("正在下载分片 %d/%d (范围: %d-%d)...\n", i+1, totalChunks, start, end)
		_, err := downloadChunk(ctx, config.URL, config.OutputPath, start, end, fileSize, config.RetryCount, config.AuthToken)
		if err != nil {
			cancel()
			return fmt.Errorf("分片下载错误: %v", err)
		}
	}

	// 验证最终文件大小
	finalSize, err := getDownloadedSize(config.OutputPath)
	if err != nil {
		return fmt.Errorf("验证文件大小失败: %v", err)
	}

	if finalSize != fileSize {
		return fmt.Errorf("下载不完整，期望大小: %d, 实际大小: %d", fileSize, finalSize)
	}

	fmt.Println("\n文件下载完成!")
	return nil
}

func main() {
	// 示例配置 - 包含AuthToken
	config := DownloadConfigRange{
		URL:        "https://example.com/large-file.iso", // 替换为实际的大文件URL
		OutputPath: "./large-file.iso",                   // 本地保存路径
		ChunkSize:  5 * 1024 * 1024,                      // 5MB分片
		Timeout:    60 * time.Second,                     // 超时时间
		RetryCount: 3,                                    // 重试次数
		AuthToken:  "your-auth-token-here",               // 替换为实际的授权令牌
	}

	// 执行下载
	if err := DownloadFileRange(config); err != nil {
		fmt.Printf("下载失败: %v\n", err)
	}
}
