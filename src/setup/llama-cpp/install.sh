#!/usr/bin/env bash
set -e

echo "🔍 检测系统环境..."
OS="$(uname -s)"

install_windows() {
    if command -v winget >/dev/null 2>&1; then
        echo "📦 检测到 winget，使用 winget 安装 llama.cpp..."
        winget install llama.cpp
    else
        echo "❌ 未检测到 winget，请先安装 Windows 包管理器 Winget："
        echo "   https://learn.microsoft.com/zh-cn/windows/package-manager/winget/"
        exit 1
    fi
}

install_mac() {
    if command -v brew >/dev/null 2>&1; then
        echo "🍺 检测到 Homebrew，使用 Homebrew 安装 llama.cpp..."
        brew install llama.cpp
    elif command -v port >/dev/null 2>&1; then
        echo "📦 检测到 MacPorts，使用 MacPorts 安装 llama.cpp..."
        sudo port install llama.cpp
    elif command -v nix >/dev/null 2>&1; then
        echo "❄️ 检测到 Nix，使用 Nix 安装 llama.cpp..."
        nix profile install nixpkgs#llama-cpp
    else
        echo "❌ 未检测到 Homebrew、MacPorts 或 Nix，请先安装其中之一再运行脚本。"
        exit 1
    fi
}

install_linux() {
    if command -v brew >/dev/null 2>&1; then
        echo "🍺 检测到 Homebrew，使用 Homebrew 安装 llama.cpp..."
        brew install llama.cpp
    elif command -v nix >/dev/null 2>&1; then
        echo "❄️ 检测到 Nix，使用 Nix 安装 llama.cpp..."
        nix profile install nixpkgs#llama-cpp
    else
        echo "❌ 未检测到 Homebrew 或 Nix，请先安装其中之一再运行脚本。"
        exit 1
    fi
}

case "$OS" in
    Darwin)
        echo "🖥 系统: macOS"
        install_mac
        ;;
    Linux)
        if grep -qi microsoft /proc/version 2>/dev/null; then
            echo "🪟 检测到 WSL 环境 (Windows Subsystem for Linux)"
            install_linux
        else
            echo "🐧 系统: Linux"
            install_linux
        fi
        ;;
    MINGW*|MSYS*|CYGWIN*)
        echo "🪟 系统: Windows (Git Bash/MSYS/Cygwin)"
        install_windows
        ;;
    *)
        echo "❌ 未知系统类型: $OS"
        exit 1
        ;;
esac

echo "✅ llama.cpp 安装完成"
