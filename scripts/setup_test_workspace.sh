#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BIN="$PROJECT_ROOT/tagfile.out"
WORKSPACE="$PROJECT_ROOT/temp/test_workspace"
IMPORTS="$PROJECT_ROOT/temp/test_imports"

cd "$PROJECT_ROOT"

if [ ! -f "$BIN" ]; then
	echo "=== 构建 tagfile ==="
	go build -o "$BIN" ./cmd/cli
fi

echo "=== 清理旧数据 ==="
rm -rf "$WORKSPACE" "$IMPORTS"
mkdir -p "$WORKSPACE"

echo "=== 生成测试文件 ($WORKSPACE) ==="
echo "dummy video content" > "$WORKSPACE/cat_british_playing.mp4"
echo "dummy video content" > "$WORKSPACE/dog_golden_outdoor.mp4"
echo "dummy video content" > "$WORKSPACE/scenery_indoor.mp4"
echo "dummy video content" > "$WORKSPACE/cat_ragdoll_playing.mp4"
echo "dummy video content" > "$WORKSPACE/dog_husky_resting.mp4"
echo "dummy video content" > "$WORKSPACE/scenery_outdoor.mp4"
echo "dummy video content" > "$WORKSPACE/cat_british_outdoor.mp4"
echo "dummy image content" > "$WORKSPACE/landscape_sunset.jpg"
echo "dummy image content" > "$WORKSPACE/portrait_night.png"

echo "9 个工作区文件已创建"
ls "$WORKSPACE"

echo ""
echo "=== 生成外部导入文件 ($IMPORTS) ==="
mkdir -p "$IMPORTS"
echo "external video from camera" > "$IMPORTS/external_cat_video.mp4"
echo "external photo from phone"   > "$IMPORTS/external_photo.jpg"
echo "2 个外部文件已创建"
ls "$IMPORTS"

echo ""
echo "=== 进入工作区，初始化数据库 ==="
cd "$WORKSPACE"
rm -rf .tagfile

run() {
	"$BIN" "$@"
}

echo "=== 创建分类项 ==="
run category add "动物类型"
run category add "场景内容"
run category add "文件类型"

echo "=== 创建标签 ==="
run tag add "动物类型" "猫"
run tag add "动物类型" "英短" --parent "猫"
run tag add "动物类型" "布偶猫" --parent "猫"
run tag add "动物类型" "狗"
run tag add "动物类型" "金毛" --parent "狗"
run tag add "动物类型" "哈士奇" --parent "狗"
run tag add "场景内容" "风景"
run tag add "场景内容" "户外"
run tag add "场景内容" "室内"
run tag add "场景内容" "玩耍"
run tag add "场景内容" "休息"
run tag add "文件类型" "视频"
run tag add "文件类型" "图片"

echo "=== 扫描工作区文件 ==="
run file scan .

echo "=== 给文件打标 ==="
run file tag cat_british_playing.mp4      1
run file tag cat_british_playing.mp4      2
run file tag cat_british_playing.mp4      10
run file tag cat_british_playing.mp4      12

run file tag dog_golden_outdoor.mp4       4
run file tag dog_golden_outdoor.mp4       5
run file tag dog_golden_outdoor.mp4       8
run file tag dog_golden_outdoor.mp4       12

run file tag scenery_indoor.mp4           7
run file tag scenery_indoor.mp4           9
run file tag scenery_indoor.mp4           12

run file tag cat_ragdoll_playing.mp4      1
run file tag cat_ragdoll_playing.mp4      3
run file tag cat_ragdoll_playing.mp4      10
run file tag cat_ragdoll_playing.mp4      12

run file tag dog_husky_resting.mp4        4
run file tag dog_husky_resting.mp4        6
run file tag dog_husky_resting.mp4        11
run file tag dog_husky_resting.mp4        12

run file tag scenery_outdoor.mp4          7
run file tag scenery_outdoor.mp4          8
run file tag scenery_outdoor.mp4          12

run file tag cat_british_outdoor.mp4      1
run file tag cat_british_outdoor.mp4      2
run file tag cat_british_outdoor.mp4      8
run file tag cat_british_outdoor.mp4      12

run file tag landscape_sunset.jpg         13
run file tag portrait_night.png           13

echo ""
echo "=== 导入外部文件 ==="
run file import "$IMPORTS/external_cat_video.mp4"
run file import "$IMPORTS/external_photo.jpg"

echo ""
echo "=== 给导入的文件打标 ==="
run file tag external_cat_video.mp4 1
run file tag external_cat_video.mp4 12
run file tag external_photo.jpg    13

echo ""
echo "========================================="
echo "              测试工作区已就绪"
echo "========================================="
echo ""
echo "分类项:"
run category list

echo ""
echo "所有标签:"
run tag list --tree

echo ""
echo "所有文件:"
run file list

echo ""
echo "=== 示例搜索 ==="
echo ""
echo "1) 所有猫视频 (AND):"
run search tag 猫 视频 --and

echo ""
echo "2) 猫 + 玩耍 (AND):"
run search tag 猫 玩耍 --and

echo ""
echo "3) 英短 + 户外 (AND):"
run search tag 英短 户外 --and

echo ""
echo "4) 所有图片文件 (排除视频):"
run search tag 图片

echo ""
echo "=== 工作区路径 ==="
echo "  $WORKSPACE"
echo "=== 外部导入源 ==="
echo "  $IMPORTS"
echo ""
echo "进入工作区使用:  cd $WORKSPACE && $BIN help"
