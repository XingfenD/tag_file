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
echo "dummy video content" > "$WORKSPACE/cos_remu_dance.mp4"
echo "dummy video content" > "$WORKSPACE/jk_uniform_interview.mp4"
echo "dummy video content" > "$WORKSPACE/casual_daily_vlog.mp4"
echo "dummy video content" > "$WORKSPACE/cos_miku_dance.mp4"
echo "dummy video content" > "$WORKSPACE/jk_sailor_daily.mp4"
echo "dummy video content" > "$WORKSPACE/casual_interview.mp4"
echo "dummy video content" > "$WORKSPACE/cos_remu_interview.mp4"
echo "dummy image content" > "$WORKSPACE/landscape_sunset.jpg"
echo "dummy image content" > "$WORKSPACE/portrait_night.png"

echo "9 个工作区文件已创建"
ls "$WORKSPACE"

echo ""
echo "=== 生成外部导入文件 ($IMPORTS) ==="
mkdir -p "$IMPORTS"
echo "external video from camera" > "$IMPORTS/external_cos_video.mp4"
echo "external photo from phone"   > "$IMPORTS/external_photo.jpg"
echo "2 个外部文件已创建"
ls "$IMPORTS"

echo ""
echo "=== 进入工作区，初始化数据库 ==="
cd "$WORKSPACE"
rm -f .tag_file.db

run() {
	"$BIN" "$@"
}

echo "=== 创建分类项 ==="
run category add "着装类型"
run category add "具体着装"
run category add "视频内容"
run category add "文件类型"

echo "=== 创建标签 ==="
run tag add "着装类型" "cos"
run tag add "着装类型" "JK"
run tag add "着装类型" "常服"
run tag add "具体着装" "蕾姆cos服"
run tag add "具体着装" "初音cos服"
run tag add "具体着装" "JK格裙"
run tag add "具体着装" "JK水手服"
run tag add "视频内容" "跳舞"
run tag add "视频内容" "采访"
run tag add "视频内容" "vlog"
run tag add "视频内容" "日常"
run tag add "文件类型" "视频"
run tag add "文件类型" "图片"

echo "=== 扫描工作区文件 ==="
run file scan .

echo "=== 给文件打标 ==="
run file tag cos_remu_dance.mp4       cos
run file tag cos_remu_dance.mp4       蕾姆cos服
run file tag cos_remu_dance.mp4       跳舞
run file tag cos_remu_dance.mp4       视频

run file tag jk_uniform_interview.mp4 JK
run file tag jk_uniform_interview.mp4 JK格裙
run file tag jk_uniform_interview.mp4 采访
run file tag jk_uniform_interview.mp4 视频

run file tag casual_daily_vlog.mp4    常服
run file tag casual_daily_vlog.mp4    vlog
run file tag casual_daily_vlog.mp4    视频

run file tag cos_miku_dance.mp4       cos
run file tag cos_miku_dance.mp4       初音cos服
run file tag cos_miku_dance.mp4       跳舞
run file tag cos_miku_dance.mp4       视频

run file tag jk_sailor_daily.mp4      JK
run file tag jk_sailor_daily.mp4      JK水手服
run file tag jk_sailor_daily.mp4      日常
run file tag jk_sailor_daily.mp4      视频

run file tag casual_interview.mp4     常服
run file tag casual_interview.mp4     采访
run file tag casual_interview.mp4     视频

run file tag cos_remu_interview.mp4   cos
run file tag cos_remu_interview.mp4   蕾姆cos服
run file tag cos_remu_interview.mp4   采访
run file tag cos_remu_interview.mp4   视频

run file tag landscape_sunset.jpg     图片
run file tag portrait_night.png       图片

echo ""
echo "=== 导入外部文件 ==="
run file import "$IMPORTS/external_cos_video.mp4"
run file import "$IMPORTS/external_photo.jpg"

echo ""
echo "=== 给导入的文件打标 ==="
run file tag imports/external_cos_video.mp4 cos
run file tag imports/external_cos_video.mp4 视频
run file tag imports/external_photo.jpg    图片

echo ""
echo "========================================="
echo "              测试工作区已就绪"
echo "========================================="
echo ""
echo "分类项:"
run category list

echo ""
echo "所有标签:"
run tag list

echo ""
echo "所有文件:"
run file list

echo ""
echo "=== 示例搜索 ==="
echo ""
echo "1) 所有 cos 视频 (AND):"
run search tag cos 视频 --and

echo ""
echo "2) cos + 跳舞 (AND):"
run search tag cos 跳舞 --and

echo ""
echo "3) 蕾姆cos服 + 采访 (AND):"
run search tag 蕾姆cos服 采访 --and

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
