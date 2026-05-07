#!/bin/bash

# 1. 基础配置
APP_NAME="drun" # 请修改为你的正式程序名称
VERSION="1.0.0"
DIST_DIR="./dist"

# 极致静态化标志：减小体积 + 强制外部静态链接(防 lib 缺失)
LDFLAGS="-s -w -extldflags '-static'"

mkdir -p $DIST_DIR

# 2. 定义核心打包函数
# 参数顺序: <OS> <ARCH> <ARM版本(可选)> <自定义输出文件名(可选)> <额外LDFLAGS(可选)>
build_target() {
    local os=$1
    local arch=$2
    local arm=$3
    local custom_name=$4
    local extra_ld=$5

    # 拼接平台标识 (如 linux_amd64, linux_armv7)
    local platform="${os}_${arch}"
    if [ -n "$arm" ]; then platform="${platform}v${arm}"; fi
    
    # 确定最终输出的二进制文件名
    local output_name
    if [ -n "$custom_name" ]; then
        output_name="$custom_name"
    else
        output_name="${APP_NAME}"
        if [ "$os" == "windows" ]; then
            output_name="${output_name}.exe"
        fi
    fi

    # 确定压缩包的基础名称 (去掉 .exe 后缀)
    local base_name="${output_name%.*}"
    local archive_name="${base_name}_${VERSION}_${platform}"

    echo "--- Building $platform -> $output_name ---"

    # 执行编译 (CGO_ENABLED=0 保证无依赖运行)
    CGO_ENABLED=0 GOOS=$os GOARCH=$arch GOARM=$arm \
    go build -ldflags "$LDFLAGS $extra_ld" -o "${DIST_DIR}/${output_name}"

    # 3. 自动压缩处理
    pushd $DIST_DIR > /dev/null
    
    if [ "$os" == "windows" ]; then
        # Windows 优先使用 zip
        if command -v zip >/dev/null 2>&1; then
            zip -q "${archive_name}.zip" "${output_name}"
            echo "Packed: ${archive_name}.zip"
        else
            tar -czf "${archive_name}.tar.gz" "${output_name}"
            echo "Packed (fallback tar): ${archive_name}.tar.gz"
        fi
    else
        # Linux/Mac 使用 tar.gz
        tar -czf "${archive_name}.tar.gz" "${output_name}"
        echo "Packed: ${archive_name}.tar.gz"
    fi

    # 删除未压缩的原始二进制文件，只保留压缩包
    rm "${output_name}"
    
    popd > /dev/null
}

# 4. 执行批量任务
echo "Starting build process..."
# 清理旧文件
rm -rf ${DIST_DIR}/*

# ================= Linux 系列 =================
# 默认 Linux AMD64
build_target "linux" "amd64" "" "" ""

# 其他 Linux 架构 (按需取消注释)
# build_target "linux" "arm" "7" "" ""
build_target "linux" "arm64" "" "" ""


# ================= Windows 系列 =================
# 调试版 (带控制台黑窗口，自定义输出文件名为 drun_debug.exe)
build_target "windows" "amd64" "" "${APP_NAME}_debug.exe" ""

# 正式版 (隐藏控制台黑窗口，使用默认名称 myapp.exe)
build_target "windows" "amd64" "" "" "-H=windowsgui"


echo "---------------------------------------"
echo "All builds completed! Files in $DIST_DIR:"
ls -lh $DIST_DIR