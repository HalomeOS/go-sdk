# go-sdk

A Go SDK for interacting with HalomeOS services, providing utilities for file uploads, downloads, and token management.

## 功能特点
- 文件上传功能 (支持断点续传)
- 文件下载 (支持断点续传，支持直接以流的方式输出，不占用本地空间，直接转发到前端)
- 身份验证令牌管理

## 安装
使用Go模块进行安装：

```bash
go get github.com/HalomeOS/go-sdk
```

## 使用示例

### 创建token
```go
package main

import (
	"fmt"
	halome "github.com/HalomeOS/go-sdk"
)

func main() {
    // account 账号
    // apiKey 密钥
    // expireTime 过期时间
    // gatewayUrl 网关地址
    token, err := halome.CreateToken(account, apiKey, expireTime, gatewayUrl)
	if err != nil {
		fmt.Printf("创建token失败: %v\n", err)
		return
	}
	fmt.Printf("token: %s\n", token)
}

```

### 文件上传
```go
package main

import (
    "fmt"
    halome "github.com/HalomeOS/go-sdk"
)

func main() {
	// 上传文件
	// 文件路径
	// token
	// 网关地址
    fileID, err := halome.UploadFile("local/file/path.txt", "your-auth-token", "https://api.halomeos.com/gateway")
    if err != nil {
        fmt.Printf("上传失败: %v\n", err)
        return
    }
    fmt.Printf("文件上传成功，ID: %s\n", fileID)
}
```

### 文件下载
```go
// 以文件流的方式下载（不保存本地，直接转发，节省本地资源,适合小文件，如图片、视频等）
// 示例使用：读取返回的流并处理
func main() {
    // 配置
        config := halome.DownloadConfig{
        URL:        "https://example.com/file.txt", // 替换为实际的文件URL
        Timeout:    60 * time.Second,               // 超时时间
        AuthToken:  "your-auth-token-here",         // 替换为实际的授权令牌
    }

    // 创建上下文
    ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
    defer cancel()

    // 下载并获取流
    resultBody, err := halome.DownloadAsStream(ctx, config)
    if err != nil {
        fmt.Printf("下载失败: %v\n", err)
        return
    }
    defer resultBody.Close() // 确保资源最终会被关闭
    //do something 
}


// 支持range协议，断点续传
func main() {
    // 示例配置 - 包含AuthToken
    config := halome.DownloadConfigRange{
        URL:        "https://example.com/large-file.iso", // 替换为实际的大文件URL
        OutputPath: "./large-file.iso",                   // 本地保存路径
        ChunkSize:  5 * 1024 * 1024,                      // 5MB分片
        Timeout:    60 * time.Second,                     // 超时时间
        AuthToken:  "your-auth-token-here",               // 替换为实际的授权令牌
    }
    
    // 执行下载
    if err := halome.DownloadFileRange(config); err != nil {
        fmt.Printf("下载失败: %v\n", err)
    }
}
```

## API文档
完整的API文档请访问 [光宇云](https://doc.halome.cc/web/#/668569011/211079820) 

## 许可证
本项目使用 Apache License 2.0 许可证 
