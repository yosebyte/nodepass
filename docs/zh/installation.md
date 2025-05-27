# 安装指南

本指南提供了使用不同方法安装 NodePass 的详细说明。选择最适合您环境和需求的安装方式。

## 系统要求

- Go 1.24或更高版本(从源代码构建时需要)
- 服务器和客户端端点之间的网络连接
- 绑定1024以下端口可能需要管理员权限

## 安装方法

### 方式1：预编译二进制文件

开始使用 NodePass 的最简单方法是为您的平台下载预编译的二进制文件。

1. 访问 GitHub 上的[发布页面](https://github.com/yosebyte/nodepass/releases)
2. 下载适合您操作系统的二进制文件(Windows、macOS、Linux)
3. 如有必要，解压缩档案
4. 使二进制文件可执行(Linux/macOS)：
   ```bash
   chmod +x nodepass
   ```
5. 将二进制文件移动到PATH中的位置：
   - Linux/macOS：`sudo mv nodepass /usr/local/bin/`
   - Windows：将位置添加到PATH环境变量

### 方式2：使用Go安装

如果您的系统上已安装Go，可以使用`go install`命令：

```bash
go install github.com/yosebyte/nodepass/cmd/nodepass@latest
```

此命令下载源代码，编译它，并将二进制文件安装到您的Go bin目录中(通常是`$GOPATH/bin`)。

### 方式3：从源代码构建

对于最新的开发版本或自定义构建：

```bash
# 克隆仓库
git clone https://github.com/yosebyte/nodepass.git

# 导航到项目目录
cd nodepass

# 构建二进制文件
go build -o nodepass ./cmd/nodepass

# 可选：安装到GOPATH/bin
go install ./cmd/nodepass
```

### 方式4：使用容器镜像

NodePass在GitHub容器注册表中提供容器镜像，非常适合容器化环境：

```bash
# 拉取容器镜像
docker pull ghcr.io/yosebyte/nodepass:latest

# 服务器模式运行
docker run -d --name nodepass-server -p 10101:10101 -p 8080:8080 \
  ghcr.io/yosebyte/nodepass server://0.0.0.0:10101/0.0.0.0:8080

# 客户端模式运行
docker run -d --name nodepass-client \
  -e MIN_POOL_CAPACITY=32 \
  -e MAX_POOL_CAPACITY=512 \
  -p 8080:8080 \
  ghcr.io/yosebyte/nodepass client://nodepass-server:10101/127.0.0.1:8080
```

### 方式5：使用管理脚本(仅限Linux)

对于Linux系统，我们提供了一键脚本：

```bash
bash <(curl -sSL https://run.nodepass.eu/np.sh)
```

- 本脚本提供了简单易用的 master 模式，即 API 模式的安装、配置和管理功能。
- 详情请参阅[https://github.com/NodePassProject/npsh](https://github.com/NodePassProject/npsh)

## 验证安装

安装后，通过检查版本来验证NodePass是否正确安装：

```bash
nodepass
```

## 下一步

安装NodePass后，您可以：

- 了解基本[使用方法](/docs/zh/usage.md)
- 探索[配置选项](/docs/zh/configuration.md)
- 尝试一些[使用示例](/docs/zh/examples.md)

## 安装问题故障排除

如果在安装过程中遇到任何问题：

- 确保您的系统满足最低要求
- 检查是否具有安装软件的正确权限
- 对于Go相关问题，使用`go version`验证您的Go安装
- 对于容器相关问题，确保Docker正确安装并运行
- 查看我们的[故障排除指南](/docs/zh/troubleshooting.md)获取更多帮助
