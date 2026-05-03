$ErrorActionPreference = "Stop"

$ScriptDir = $PSScriptRoot
$ProjectRoot = Split-Path -Parent $ScriptDir
$Bin = Join-Path $ProjectRoot "tagfile.exe"
$Workspace = Join-Path $ProjectRoot "temp\test_workspace"
$Imports = Join-Path $ProjectRoot "temp\test_imports"

Set-Location $ProjectRoot

if (-not (Test-Path $Bin)) {
    Write-Host "=== 构建 tagfile ==="
    go build -o $Bin ./cmd/cli
}

Write-Host "=== 清理旧数据 ==="
if (Test-Path $Workspace) { Remove-Item -Recurse -Force $Workspace }
if (Test-Path $Imports) { Remove-Item -Recurse -Force $Imports }
New-Item -ItemType Directory -Force -Path $Workspace | Out-Null

Write-Host "=== 生成测试文件 ($Workspace) ==="
Set-Content -Path "$Workspace\cat_british_playing.mp4" -Value "dummy video content"
Set-Content -Path "$Workspace\dog_golden_outdoor.mp4" -Value "dummy video content"
Set-Content -Path "$Workspace\scenery_indoor.mp4" -Value "dummy video content"
Set-Content -Path "$Workspace\cat_ragdoll_playing.mp4" -Value "dummy video content"
Set-Content -Path "$Workspace\dog_husky_resting.mp4" -Value "dummy video content"
Set-Content -Path "$Workspace\scenery_outdoor.mp4" -Value "dummy video content"
Set-Content -Path "$Workspace\cat_british_outdoor.mp4" -Value "dummy video content"
Set-Content -Path "$Workspace\landscape_sunset.jpg" -Value "dummy image content"
Set-Content -Path "$Workspace\portrait_night.png" -Value "dummy image content"

Write-Host "9 个工作区文件已创建"
Get-ChildItem $Workspace

Write-Host ""
Write-Host "=== 生成外部导入文件 ($Imports) ==="
New-Item -ItemType Directory -Force -Path $Imports | Out-Null
Set-Content -Path "$Imports\external_cat_video.mp4" -Value "external video from camera"
Set-Content -Path "$Imports\external_photo.jpg" -Value "external photo from phone"
Write-Host "2 个外部文件已创建"
Get-ChildItem $Imports

Write-Host ""
Write-Host "=== 进入工作区，初始化数据库 ==="
Set-Location $Workspace
Remove-Item -Force .tag_file.db -ErrorAction SilentlyContinue

function Run {
    & $Bin @args
    if ($LASTEXITCODE -ne 0) {
        throw "命令失败: $Bin $args"
    }
}

Write-Host "=== 创建分类项 ==="
Run category add "动物类型"
Run category add "具体品种"
Run category add "场景内容"
Run category add "文件类型"

Write-Host "=== 创建标签 ==="
Run tag add "动物类型" "猫"
Run tag add "动物类型" "狗"
Run tag add "动物类型" "风景"
Run tag add "具体品种" "英短"
Run tag add "具体品种" "布偶猫"
Run tag add "具体品种" "金毛"
Run tag add "具体品种" "哈士奇"
Run tag add "场景内容" "玩耍"
Run tag add "场景内容" "户外"
Run tag add "场景内容" "室内"
Run tag add "场景内容" "休息"
Run tag add "文件类型" "视频"
Run tag add "文件类型" "图片"

Write-Host "=== 扫描工作区文件 ==="
Run file scan .

Write-Host "=== 给文件打标 ==="
Run file tag cat_british_playing.mp4      猫
Run file tag cat_british_playing.mp4      英短
Run file tag cat_british_playing.mp4      玩耍
Run file tag cat_british_playing.mp4      视频

Run file tag dog_golden_outdoor.mp4       狗
Run file tag dog_golden_outdoor.mp4       金毛
Run file tag dog_golden_outdoor.mp4       户外
Run file tag dog_golden_outdoor.mp4       视频

Run file tag scenery_indoor.mp4           风景
Run file tag scenery_indoor.mp4           室内
Run file tag scenery_indoor.mp4           视频

Run file tag cat_ragdoll_playing.mp4      猫
Run file tag cat_ragdoll_playing.mp4      布偶猫
Run file tag cat_ragdoll_playing.mp4      玩耍
Run file tag cat_ragdoll_playing.mp4      视频

Run file tag dog_husky_resting.mp4        狗
Run file tag dog_husky_resting.mp4        哈士奇
Run file tag dog_husky_resting.mp4        休息
Run file tag dog_husky_resting.mp4        视频

Run file tag scenery_outdoor.mp4          风景
Run file tag scenery_outdoor.mp4          户外
Run file tag scenery_outdoor.mp4          视频

Run file tag cat_british_outdoor.mp4      猫
Run file tag cat_british_outdoor.mp4      英短
Run file tag cat_british_outdoor.mp4      户外
Run file tag cat_british_outdoor.mp4      视频

Run file tag landscape_sunset.jpg         图片
Run file tag portrait_night.png           图片

Write-Host ""
Write-Host "=== 导入外部文件 ==="
Run file import "$Imports\external_cat_video.mp4"
Run file import "$Imports\external_photo.jpg"

Write-Host ""
Write-Host "=== 给导入的文件打标 ==="
Run file tag external_cat_video.mp4 猫
Run file tag external_cat_video.mp4 视频
Run file tag external_photo.jpg    图片

Write-Host ""
Write-Host "========================================="
Write-Host "              测试工作区已就绪"
Write-Host "========================================="
Write-Host ""
Write-Host "分类项:"
Run category list

Write-Host ""
Write-Host "所有标签:"
Run tag list

Write-Host ""
Write-Host "所有文件:"
Run file list

Write-Host ""
Write-Host "=== 示例搜索 ==="
Write-Host ""
Write-Host "1) 所有猫视频 (AND):"
Run search tag 猫 视频 --and

Write-Host ""
Write-Host "2) 猫 + 玩耍 (AND):"
Run search tag 猫 玩耍 --and

Write-Host ""
Write-Host "3) 英短 + 户外 (AND):"
Run search tag 英短 户外 --and

Write-Host ""
Write-Host "4) 所有图片文件 (排除视频):"
Run search tag 图片

Write-Host ""
Write-Host "=== 工作区路径 ==="
Write-Host "  $Workspace"
Write-Host "=== 外部导入源 ==="
Write-Host "  $Imports"
Write-Host ""
Write-Host "进入工作区使用:  cd $Workspace ; $Bin help"
