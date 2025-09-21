package go_sdk

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type RespUploadResp struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	FileIndex int64  `json:"fileIndex"`
	Id        string `json:"id"`
}

// 文件上传
// FilePath: 文件路径
// AuthToken: 认证token
// GatewayUrl: 网关url
func UploadFile(FilePath, AuthToken, GatewayUrl string) (id string, err error) {
	//resp := &RespUploadResp{}
	// 切片大小，这里设置为2MB
	const chunkSize = 2 * 1024 * 1024
	// 打开文件
	file, err := os.Open(FilePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()
	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("获取文件信息失败: %v", err)
	}
	FileName := fileInfo.Name()
	FileSize := fileInfo.Size()
	FileStartIndex := "0"
	// 计算文件MD5
	FileMd5, err := getFileMD5(FilePath)
	if err != nil {
		return "", fmt.Errorf("计算文件MD5失败: %v", err)
	}
	//// 计算总切片数
	//totalChunks := int(math.Ceil(float64(FileSize) / float64(chunkSize)))
	//fmt.Printf("文件大小: %d bytes, 切片大小: %d bytes, 总切片数: %d\n", FileSize, chunkSize, totalChunks)
	var offset int64 = 0
	for {
		//切割文件
		chunkData, err := readFileBytes(file, offset, chunkSize)
		if err != nil {
			return "", err
		}
		// 创建请求
		req, err := http.NewRequest("POST", GatewayUrl+"/v1/addLargeFile", bytes.NewBuffer(chunkData))
		if err != nil {
			return "", err
		}

		// 设置头信息
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("FileStartIndex", FileStartIndex)
		req.Header.Set("FileSize", strconv.FormatInt(FileSize, 10))
		req.Header.Set("FileName", FileName)
		req.Header.Set("FileMd5", FileMd5)
		req.Header.Set("AuthToken", AuthToken)

		// 创建客户端并设置超时
		client := &http.Client{
			Timeout: 24 * time.Hour,
		}

		// 发送请求
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		// 读取响应
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("上传失败, 状态码: %d", resp.StatusCode)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("读取响应失败: %v", err)
		}
		var respUploadResp RespUploadResp
		err = json.Unmarshal(body, &respUploadResp)
		if err != nil {
			return "", fmt.Errorf("解析响应失败: %v", err)
		}
		//断点续传
		if respUploadResp.Code == 7 {
			// 修正当前偏移量
			offset = respUploadResp.FileIndex
			FileStartIndex = strconv.FormatInt(offset, 10)
			continue
		}

		if respUploadResp.Code != 200 {
			return "", fmt.Errorf("上传失败, 状态码: %d, 消息: %s", respUploadResp.Code, respUploadResp.Message)
		}

		id = respUploadResp.Id
		//上传成功
		if id != "" {
			return id, nil
		}
		// 计算下一个偏移量
		offset = respUploadResp.FileIndex
		FileStartIndex = strconv.FormatInt(offset, 10)
	}
}

// 计算文件MD5，用于标识文件唯一性
func getFileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// 读取文件指定范围的字节
// offset: 起始位置（从0开始）
// length: 要读取的字节数
func readFileBytes(file *os.File, offset int64, length int) ([]byte, error) {
	// 创建缓冲区
	buffer := make([]byte, length)
	// 从指定位置读取指定长度的字节
	n, err := file.ReadAt(buffer, offset)
	if err != nil {
		// EOF是正常文件结束，不是错误
		if err == io.EOF {
			// 返回实际读取的字节
			return buffer[:n], nil
		}
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}
	// 返回实际读取的字节（可能小于请求的长度，如到达文件末尾）
	return buffer[:n], nil
}
