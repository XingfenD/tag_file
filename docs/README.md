# tagfile

跨平台文件整理与检索命令行工具。对指定目录内的文件进行可配置的分类打标，支持快速标签检索。

## 安装

```bash
git clone https://github.com/XingfenD/tag_file.git
cd tag_file
make install
```

或直接编译：

```bash
go build -o tagfile.out ./cmd/cli
```

## 快速开始

在一个目录中初始化工作区并创建演示数据：

```bash
make setup
```

然后进入工作区：

```bash
cd temp/test_workspace
```

## 使用流程

### 1. 创建分类项

分类项是对标签的分组，如"着装类型"、"视频内容"等。

```bash
tagfile category add 着装类型
tagfile category add 视频内容
tagfile category add 文件类型

tagfile category list
```

### 2. 创建标签

标签归属于分类项，同一个分类项下标签名需要唯一。

```bash
tagfile tag add 着装类型 cos
tagfile tag add 着装类型 JK
tagfile tag add 视频内容 跳舞
tagfile tag add 视频内容 采访
tagfile tag add 文件类型 视频
tagfile tag add 文件类型 图片

tagfile tag list              # 列出所有标签
tagfile tag list 着装类型      # 按分类项过滤
```

### 3. 扫描文件

将目录中的文件注册到数据库。

```bash
tagfile file scan .            # 扫描当前目录
tagfile file scan /path/to/dir # 扫描指定目录

tagfile file list              # 查看已注册文件
```

### 4. 给文件打标签

```bash
tagfile file tag video1.mp4 cos
tagfile file tag video1.mp4 跳舞
tagfile file tag video1.mp4 视频

tagfile file info video1.mp4   # 查看文件详情和标签
tagfile file untag video1.mp4 跳舞  # 移除标签
```

### 5. 搜索文件

按标签搜索文件，支持 AND/OR/NOT 逻辑。

```bash
# OR 搜索（默认）：文件拥有任一标签
tagfile search tag cos JK

# AND 搜索：文件必须同时拥有所有标签
tagfile search tag --and cos 跳舞

# NOT 搜索：排除拥有指定标签的文件
tagfile search tag --not 常服

# 组合：包含 JK 但排除 JK格裙
tagfile search tag JK --not JK格裙
```

### 6. 导入外部文件

从其他位置导入文件到当前目录。

```bash
tagfile file import /path/to/external/video.mp4
```

### 7. 属性管理（高级）

为标签定义更细粒度的属性。

```bash
# 为标签添加属性
tagfile attribute add cos 角色

# 为属性添加可选值
tagfile attribute value add 1 蕾姆
tagfile attribute value add 1 初音未来

# 查看属性和值
tagfile attribute list cos
tagfile attribute value list 1
```

### 8. 配置管理

```bash
tagfile config list
tagfile config set mykey myvalue
tagfile config get mykey
tagfile config unset mykey
```

## 命令总览

```
tagfile help

  category   管理分类项
    add      新增分类项
    remove   删除分类项
    list     列出所有分类项

  tag        管理标签
    add      新增标签
    remove   删除标签
    list     列出标签

  file       管理文件
    scan     扫描目录注册文件
    import   从外部引入文件
    list     列出已注册文件
    tag      给文件打标签
    untag    移除文件标签
    info     查看文件详情
    search   按文件名搜索

  search     按标签搜索文件
    tag      按标签搜索 (--and / --or / --not)

  attribute  管理标签属性
    add      为标签添加属性定义
    remove   删除属性
    list     列出属性
    value    管理属性值
      add    添加属性值
      remove 删除属性值
      list   列出属性值

  config     管理配置项
    set      设置配置项
    get      查看配置项
    list     列出所有配置
    unset    删除配置项
```

## 工作原理

每个目录独立一个 `.tag_file.db`（SQLite），文件只索引不移动（外部导入时复制到当前目录）。

```bash
# 在你想要管理的目录中
cd ~/my_videos
tagfile category add 着装类型
tagfile tag add 着装类型 cos
tagfile file scan .
tagfile file tag video.mp4 cos
tagfile search tag cos
```

## 开发

```bash
make build    # 编译
make test     # 运行测试 (47 个)
make setup    # 创建演示工作区
make clean    # 清理
make install  # 安装到 GOPATH
```

测试还包括标准 Go 测试：

```bash
go test ./internal/service/... -v
```
