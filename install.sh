#!/usr/bin/env bash
# 一键安装脚本：编译二进制到 PATH + 安装 Skill 到 Claude Code
# 用法：在项目根目录执行 ./install.sh
set -euo pipefail

# 切到脚本所在目录（项目根），保证相对路径可靠
cd "$(dirname "$0")"

BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
BIN_NAME="xiaoyuzhoufm-mcp"
SKILL_SRC="skill/xiaoyuzhoufm"
SKILL_DST="${SKILL_DST:-$HOME/.claude/skills/xiaoyuzhoufm}"

echo "==> 1/3 编译二进制"
if ! command -v go >/dev/null 2>&1; then
  echo "错误：未找到 go，请先安装 Go (https://go.dev/dl/) 后重试。" >&2
  exit 1
fi
mkdir -p "$BIN_DIR"
go build -o "$BIN_DIR/$BIN_NAME" ./cmd/xiaoyuzhoufm-mcp
echo "    已安装到 $BIN_DIR/$BIN_NAME"

echo "==> 2/3 检查 PATH"
case ":$PATH:" in
  *":$BIN_DIR:"*)
    echo "    $BIN_DIR 已在 PATH 中 ✓" ;;
  *)
    echo "    警告：$BIN_DIR 不在 PATH 中。" >&2
    echo "    请把下面这行加入 ~/.zshrc 或 ~/.bashrc 后重开终端：" >&2
    echo "        export PATH=\"$BIN_DIR:\$PATH\"" >&2 ;;
esac

echo "==> 3/3 安装 Skill (供 Claude Code 使用)"
if [ -d "$SKILL_SRC" ]; then
  mkdir -p "$(dirname "$SKILL_DST")"
  rm -rf "$SKILL_DST"
  cp -R "$SKILL_SRC" "$SKILL_DST"
  echo "    Skill 已安装到 $SKILL_DST"
else
  echo "    跳过：未找到 $SKILL_SRC（如不使用 Claude Code Skill 可忽略）"
fi

echo ""
echo "完成！接下来："
echo "  1. 登录一次（首次必做）: $BIN_NAME init"
echo "  2. 验证:                 $BIN_NAME cli search-podcast --keyword 纵横四海 --brief"
