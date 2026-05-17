package file

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/XingfenD/tag_file/cmd/cli/commands/util"
	repoTag "github.com/XingfenD/tag_file/internal/repository/tag"
	svcfile "github.com/XingfenD/tag_file/internal/service/file"
	svctaggroup "github.com/XingfenD/tag_file/internal/service/taggroup"
	"github.com/spf13/cobra"
)

func resolveTagID(input string) (uint, error) {
	id, err := strconv.ParseUint(input, 10, 64)
	if err == nil {
		return uint(id), nil
	}
	tagRepo := repoTag.NewRepository()
	tags, err := tagRepo.Find().ByName(input).Preload("Category").All()
	if err != nil {
		return 0, fmt.Errorf("查询标签失败: %w", err)
	}
	switch len(tags) {
	case 0:
		return 0, fmt.Errorf("标签 '%s' 不存在", input)
	case 1:
		return tags[0].TagID, nil
	default:
		lines := make([]string, 0, len(tags)+1)
		lines = append(lines, fmt.Sprintf("标签 '%s' 存在多个匹配:", input))
		for _, t := range tags {
			lines = append(lines, fmt.Sprintf("  ID=%-4d 分类=%s", t.TagID, t.Category.CategoryName))
		}
		lines = append(lines, "请使用标签ID指定")
		return 0, fmt.Errorf(strings.Join(lines, "\n"))
	}
}

var FileCmd = &cobra.Command{
	Use:   "file",
	Short: "管理文件",
	Long:  `扫描目录注册文件、导入外部文件、对文件打标或取消标签。`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var fileScanCmd = &cobra.Command{
	Use:   "scan [目录]",
	Short: "扫描目录注册文件",
	Long:  `扫描指定目录，将新文件注册到数据库。默认增量扫描，--full 强制全量。`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		full, _ := cmd.Flags().GetBool("full")
		svc := svcfile.NewService()
		result, err := svc.ScanDirectoryOptions(dir, full)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("扫描目录: %s\n", result.Root)
		fmt.Printf("  遍历目录: %d\n", result.DirsScanned)
		fmt.Printf("  发现文件: %d\n", result.FilesFound)
		fmt.Printf("  新增注册: %d\n", result.FilesAdded)
		for _, e := range result.Errors {
			fmt.Fprintln(os.Stderr, "  警告:", e)
		}
	},
}

var fileImportCmd = &cobra.Command{
	Use:   "import <外部路径>",
	Short: "从外部引入文件",
	Long:  `将外部文件复制到当前管理的目录中并注册。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc := svcfile.NewService()
		file, err := svc.ImportFile(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("文件已导入: %s\n", file.FilePath)
	},
}

var fileListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出已注册文件",
	Long:  `列出当前数据库中所有已注册的文件。支持 --limit/--offset 分页。`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		svc := svcfile.NewService()
		list, total, err := svc.ListFiles(offset, limit)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		if len(list) == 0 {
			fmt.Println("暂无已注册文件")
			return
		}
		fmt.Printf("%-4s %-40s %-10s %s\n", "ID", "路径", "大小", "类型")
		for _, f := range list {
			sizeStr := util.FormatSize(f.FileSize)
			fmt.Printf("%-4d %-40s %-10s %s\n", f.FileID, util.Truncate(f.FilePath, 40), sizeStr, f.FileType)
		}
		if limit > 0 {
			fmt.Printf("--- %d/%d ---\n", len(list), total)
		}
	},
}

var fileTagCmd = &cobra.Command{
	Use:   "tag <文件路径> <标签名或ID>",
	Short: "给文件打标",
	Long:  `给已注册的文件添加一个标签。--value 可指定标签值。`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		tagID, err := resolveTagID(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		tagValue, _ := cmd.Flags().GetString("value")
		svc := svcfile.NewService()
		if err := svc.TagFile(filePath, tagID, tagValue); err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("已为 '%s' 添加标签 '%s'", filePath, args[1])
		if tagValue != "" {
			fmt.Printf(":%s", tagValue)
		}
		fmt.Println()
	},
}

var fileUntagCmd = &cobra.Command{
	Use:   "untag <文件路径> <标签名或ID>",
	Short: "移除文件标签",
	Long:  `移除已注册文件上的一个标签。--value 可指定标签值。`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		tagID, err := resolveTagID(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		tagValue, _ := cmd.Flags().GetString("value")
		svc := svcfile.NewService()
		if err := svc.UntagFile(filePath, tagID, tagValue); err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("已从 '%s' 移除标签 '%s'\n", filePath, args[1])
	},
}

var fileInfoCmd = &cobra.Command{
	Use:   "info <文件路径>",
	Short: "查看文件详情",
	Long:  `查看已注册文件的详细信息和关联标签。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc := svcfile.NewService()
		file, tags, valMap, err := svc.GetFileInfo(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("文件ID:   %d\n", file.FileID)
		fmt.Printf("路径:     %s\n", file.FilePath)
		fmt.Printf("大小:     %s\n", util.FormatSize(file.FileSize))
		fmt.Printf("类型:     %s\n", file.FileType)
		if len(tags) > 0 {
			fmt.Println("标签:")
			for _, t := range tags {
				vals := valMap[t.TagID]
				if len(vals) > 0 {
					for _, v := range vals {
						fmt.Printf("  [%s] %s:%s\n", t.Category.CategoryName, t.TagName, v)
					}
				} else {
					fmt.Printf("  [%s] %s\n", t.Category.CategoryName, t.TagName)
				}
			}
		} else {
			fmt.Println("标签:     无")
		}
	},
}

var fileTagsCmd = &cobra.Command{
	Use:   "tags <文件路径>",
	Short: "列出文件的所有标签",
	Long:  `列出指定文件的所有标签（含标签值），支持 --limit/--offset 分页。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		svc := svcfile.NewService()
		list, total, err := svc.ListFileTags(args[0], offset, limit)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		if len(list) == 0 {
			fmt.Println("暂无标签")
			return
		}
		fmt.Printf("%-4s %-20s %-20s %-20s %s\n", "ID", "标签组", "分类项", "标签", "值")
		for _, ft := range list {
			tagVal := ft.TagValue
			if tagVal == "" {
				tagVal = "-"
			}
			fmt.Printf("%-4d %-20s %-20s %-20s %s\n", ft.ID, ft.Group.GroupName, ft.Tag.Category.CategoryName, ft.Tag.TagName, tagVal)
		}
		if limit > 0 {
			fmt.Printf("--- %d/%d ---\n", len(list), total)
		}
	},
}

var fileExistsCmd = &cobra.Command{
	Use:   "exists <文件路径>",
	Short: "判断文件是否已索引",
	Long:  `检查指定文件是否已在数据库中注册。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc := svcfile.NewService()
		indexed, err := svc.IsIndexed(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		if indexed {
			fmt.Printf("已索引: %s\n", args[0])
		} else {
			fmt.Printf("未索引: %s\n", args[0])
			os.Exit(1)
		}
	},
}

var fileDiffCmd = &cobra.Command{
	Use:   "diff [目录]",
	Short: "列出未索引的文件",
	Long:  `对比目录与数据库，列出存在但尚未注册的文件。默认仅当前目录，-r 递归。`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		recursive, _ := cmd.Flags().GetBool("recursive")
		svc := svcfile.NewService()
		result, err := svc.DiffUnindexed(dir, recursive)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		if recursive {
			printTree(result)
		} else {
			fmt.Printf("扫描目录: %s\n", result.Root)
			fmt.Printf("  发现文件: %d\n", result.FilesFound)
			fmt.Printf("  未索引:   %d\n", result.FilesUnindexed)
			for _, e := range result.Errors {
				fmt.Fprintln(os.Stderr, "  警告:", e)
			}
			if len(result.Unindexed) > 0 {
				fmt.Println()
				for _, p := range result.Unindexed {
					fmt.Printf("  %s\n", p)
				}
			}
		}
	},
}

func printTree(result *svcfile.DiffResult) {
	if result.FilesUnindexed == 0 {
		fmt.Println("所有文件已索引")
		return
	}
	type dirNode struct {
		path  string
		count int
		subs  map[string]*dirNode
	}
	root := &dirNode{path: result.Root, subs: make(map[string]*dirNode)}
	for _, p := range result.Unindexed {
		parts := strings.Split(strings.TrimPrefix(p, "./"), "/")
		cur := root
		for _, part := range parts {
			if cur.subs[part] == nil {
				cur.subs[part] = &dirNode{path: part, subs: make(map[string]*dirNode)}
			}
			cur.count++
			cur = cur.subs[part]
		}
		cur.count++
	}
	var walk func(n *dirNode, prefix string, isLast bool, showRoot bool)
	walk = func(n *dirNode, prefix string, isLast bool, showRoot bool) {
		if showRoot {
			fmt.Printf("%s\n", n.path)
		}
		keys := make([]string, 0, len(n.subs))
		for k := range n.subs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			child := n.subs[k]
			last := i == len(keys)-1
			branch := "├── "
			nextPrefix := "│   "
			if last {
				branch = "└── "
				nextPrefix = "    "
			}
			if len(child.subs) == 0 {
				fmt.Printf("%s%s%s\n", prefix, branch, k)
			} else {
				fmt.Printf("%s%s%s/ (%d)\n", prefix, branch, k, child.count)
				walk(child, prefix+nextPrefix, last, false)
			}
		}
	}
	if len(root.subs) == 0 {
		return
	}
	walk(root, "", true, true)
}

var fileGroupCmd = &cobra.Command{
	Use:   "group",
	Short: "管理文件标签组",
	Long:  `管理文件的多组标签。默认组自动创建。`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var fileGroupAddCmd = &cobra.Command{
	Use:   "add <文件路径> <组名>",
	Short: "新增标签组",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fileSvc := svcfile.NewService()
		file, err := fileSvc.FindByPath(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		svc := svctaggroup.NewService()
		group, err := svc.AddGroup(file.FileID, args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("标签组 '%s' 已创建 (ID: %d)\n", group.GroupName, group.GroupID)
	},
}

var fileGroupRemoveCmd = &cobra.Command{
	Use:   "remove <文件路径> <组名>",
	Short: "删除标签组",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fileSvc := svcfile.NewService()
		file, err := fileSvc.FindByPath(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		svc := svctaggroup.NewService()
		if err := svc.RemoveGroup(file.FileID, args[1]); err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("标签组 '%s' 已删除\n", args[1])
	},
}

var fileGroupListCmd = &cobra.Command{
	Use:   "list <文件路径>",
	Short: "列出标签组",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fileSvc := svcfile.NewService()
		file, err := fileSvc.FindByPath(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		svc := svctaggroup.NewService()
		groups, err := svc.ListGroups(file.FileID)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		if len(groups) == 0 {
			fmt.Println("暂无标签组")
			return
		}
		fmt.Printf("%-4s %-20s %-10s %s\n", "ID", "名称", "默认", "描述")
		for _, g := range groups {
			isDef := ""
			if g.IsDefault {
				isDef = "是"
			}
			fmt.Printf("%-4d %-20s %-10s %s\n", g.GroupID, g.GroupName, isDef, g.Description)
		}
	},
}

var fileGroupTagCmd = &cobra.Command{
	Use:   "tag <文件路径> <组名> <标签名或ID>",
	Short: "给标签组打标",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		fileSvc := svcfile.NewService()
		file, err := fileSvc.FindByPath(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		tagID, err := resolveTagID(args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		tagValue, _ := cmd.Flags().GetString("value")
		svc := svctaggroup.NewService()
		if err := svc.TagGroup(file.FileID, args[1], tagID, tagValue); err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("已为标签组 '%s' 添加标签 '%s'", args[1], args[2])
		if tagValue != "" {
			fmt.Printf(":%s", tagValue)
		}
		fmt.Println()
	},
}

var fileGroupUntagCmd = &cobra.Command{
	Use:   "untag <文件路径> <组名> <标签名或ID>",
	Short: "从标签组移除标签",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		fileSvc := svcfile.NewService()
		file, err := fileSvc.FindByPath(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		tagID, err := resolveTagID(args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		tagValue, _ := cmd.Flags().GetString("value")
		svc := svctaggroup.NewService()
		if err := svc.UntagGroup(file.FileID, args[1], tagID, tagValue); err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("已从标签组 '%s' 移除标签 '%s'\n", args[1], args[2])
	},
}

func init() {
	fileScanCmd.Flags().Bool("full", false, "强制全量扫描")
	fileListCmd.Flags().Int("limit", 0, "分页条数")
	fileListCmd.Flags().Int("offset", 0, "分页偏移")
	fileDiffCmd.Flags().BoolP("recursive", "r", false, "递归扫描子目录，树形展示")
	fileTagCmd.Flags().String("value", "", "标签值")
	fileUntagCmd.Flags().String("value", "", "标签值")
	fileTagsCmd.Flags().Int("limit", 0, "分页条数")
	fileTagsCmd.Flags().Int("offset", 0, "分页偏移")
	fileGroupTagCmd.Flags().String("value", "", "标签值")
	fileGroupUntagCmd.Flags().String("value", "", "标签值")

	FileCmd.AddCommand(fileScanCmd)
	FileCmd.AddCommand(fileDiffCmd)
	FileCmd.AddCommand(fileExistsCmd)
	FileCmd.AddCommand(fileImportCmd)
	FileCmd.AddCommand(fileListCmd)
	FileCmd.AddCommand(fileTagCmd)
	FileCmd.AddCommand(fileUntagCmd)
	FileCmd.AddCommand(fileInfoCmd)
	FileCmd.AddCommand(fileTagsCmd)
	FileCmd.AddCommand(fileGroupCmd)
	FileCmd.AddCommand(fileSearchCmd)

	fileGroupCmd.AddCommand(fileGroupAddCmd)
	fileGroupCmd.AddCommand(fileGroupRemoveCmd)
	fileGroupCmd.AddCommand(fileGroupListCmd)
	fileGroupCmd.AddCommand(fileGroupTagCmd)
	fileGroupCmd.AddCommand(fileGroupUntagCmd)
}

var fileSearchCmd = &cobra.Command{
	Use:   "search <文件名关键词>",
	Short: "按文件名搜索",
	Long:  `在已注册文件中按文件名关键词搜索。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc := svcfile.NewService()
		results, err := svc.SearchFilesWithTags(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		if len(results) == 0 {
			fmt.Println("未找到匹配的文件")
			return
		}
		for _, r := range results {
			fmt.Printf("%s (%s)\n", r.File.FilePath, util.FormatSize(r.File.FileSize))
			for _, t := range r.Tags {
				vals := r.ValMap[t.TagID]
				if len(vals) > 0 {
					for _, v := range vals {
						fmt.Printf("  [%s] %s:%s\n", t.Category.CategoryName, t.TagName, v)
					}
				} else {
					fmt.Printf("  [%s] %s\n", t.Category.CategoryName, t.TagName)
				}
			}
		}
	},
}
