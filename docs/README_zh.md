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

```bash
make setup && cd temp/test_workspace
```

## 使用流程

### 1. 创建分类项

分类项是标签的分组，如"动物类型"、"场景内容"等。名称全局唯一。

```bash
tagfile category add 动物类型
tagfile category add 场景内容
tagfile category list
```

### 2. 创建标签

标签必须归属某个分类项，同分类项下标签名唯一。

```bash
tagfile tag add 动物类型 猫
tagfile tag add 动物类型 狗
tagfile tag add 场景内容 玩耍
tagfile tag list
tagfile tag list 动物类型     # 按分类项过滤
```

### 3. 扫描文件

```bash
tagfile file scan .            # 扫描当前目录
tagfile file list              # 查看已注册文件
tagfile file info video.mp4    # 查看文件详情
```

### 4. 给文件打标签

```bash
tagfile file tag video.mp4 猫
tagfile file tag video.mp4 玩耍
tagfile file untag video.mp4 猫   # 移除标签
```

### 5. 搜索文件

```bash
# OR 搜索（默认）
tagfile search tag 猫 狗

# AND 搜索
tagfile search tag --and 猫 玩耍

# NOT 搜索（排除）
tagfile search tag --not 风景

# 组合
tagfile search tag 狗 --not 金毛
```

### 6. 导入外部文件

外部文件自动复制到当前目录并注册。

```bash
tagfile file import /path/to/video.mp4
```

### 7. 属性管理

为标签添加更细粒度的属性定义和可选值。

```bash
tagfile attribute add 猫 品种
tagfile attribute value add 1 英短
tagfile attribute value add 1 布偶猫
tagfile attribute list
tagfile attribute value list 1
```

### 8. 配置管理

```bash
tagfile config set mykey myvalue
tagfile config get mykey
tagfile config list
tagfile config unset mykey
```

## 命令参考

| 命令 | 说明 |
|---|---|
| `category {add,remove,list}` | 管理分类项 |
| `tag {add,remove,list}` | 管理标签 |
| `file {scan,import,list,tag,untag,info,search}` | 文件管理 |
| `search tag <标签> --and/--or/--not` | 标签搜索 |
| `attribute {add,remove,list}` | 属性管理 |
| `attribute value {add,remove,list}` | 属性值管理 |
| `config {set,get,list,unset}` | 配置管理 |
| `completion {bash,zsh,fish,powershell}` | 生成 shell 补全脚本 |

## 工作原理

- 每个目录独立一个 `.tag_file.db`（SQLite）
- 文件只索引不移动（外部导入时复制到当前目录）
- 分类项名称全局唯一，标签名在同分类项下唯一

## 开发

```bash
make build    # 编译
make test     # 运行全部测试
make setup    # 创建演示工作区
make clean    # 清理
```
