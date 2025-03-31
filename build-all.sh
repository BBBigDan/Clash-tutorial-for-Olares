#!/bin/bash
# 文件名: build-all.sh

# 初始化Go模块（如果尚未初始化）
if [ ! -f go.mod ]; then
    echo "初始化Go模块..."
    go mod init clash-setup
    go get gopkg.in/yaml.v3
fi

# 确保所有依赖都已下载
go mod tidy

# 创建输出目录
mkdir -p build

# 定义要编译的平台
platforms=(
  "linux/amd64"
  "linux/386"
  "linux/arm64"
  "linux/arm/7"
#   "windows/amd64"
  "darwin/amd64"
  "darwin/arm64"
)

# 编译每个平台
for platform in "${platforms[@]}"; do
  platform_split=(${platform//\// })
  GOOS=${platform_split[0]}
  GOARCH=${platform_split[1]}
  GOARM=${platform_split[2]}
  
  output_name=clash-setup
  
  # 设置输出文件名
  if [ $GOOS = "windows" ]; then
    output_name+='.exe'
  fi
  
  # 添加架构信息到文件名
  output_name+="-$GOOS-$GOARCH"
  if [ ! -z "$GOARM" ]; then
    output_name+=v$GOARM
  fi
  
  echo "编译 $output_name"
  
  # 设置环境变量并编译
  env GOOS=$GOOS GOARCH=$GOARCH GOARM=$GOARM go build -o build/$output_name
  
  if [ $? -ne 0 ]; then
    echo "编译 $platform 失败"
  else
    echo "编译 $platform 成功"
  fi
done

echo "所有架构编译完成！"
