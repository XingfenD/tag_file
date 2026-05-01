package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/XingfenD/tag_file/internal/service"
	"github.com/spf13/cobra"
)

var FileCmd = &cobra.Command{
	Use:   "file",
	Short: "管理文件",
	Long:  `扫描目录注册文件、导入外部文件、对文件打标或取消标签。`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var (
	fileScanFullFlag      bool
)

var fileScanCmd = &cobra.Command{
	Use:   "scan [目录]",
	Short: "扫描目录注册文件",
	Long:  `扫描指定目录，将新文件注册到数据库。默认增量扫描（跳过未变更文件），--full 强制全量。`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		svc := service.NewFileService()
		result, err := svc.ScanDirectoryOptions(dir, fileScanFullFlag)
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
		svc := service.NewFileService()
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
	Long:  `列出当前数据库中所有已注册的文件。`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		svc := service.NewFileService()
		files, err := svc.ListFiles()
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		if len(files) == 0 {
			fmt.Println("暂无已注册文件")
			return
		}
		fmt.Printf("%-4s %-40s %-10s %s\n", "ID", "路径", "大小", "类型")
		for _, f := range files {
			sizeStr := formatSize(f.FileSize)
			fmt.Printf("%-4d %-40s %-10s %s\n", f.FileID, truncate(f.FilePath, 40), sizeStr, f.FileType)
		}
	},
}

var fileTagCmd = &cobra.Command{
	Use:   "tag <文件路径> <标签名>",
	Short: "给文件打标",
	Long:  `给已注册的文件添加一个标签。`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		tagName := args[1]
		svc := service.NewFileService()
		if err := svc.TagFile(filePath, tagName); err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("已为 '%s' 添加标签 '%s'\n", filePath, tagName)
	},
}

var fileUntagCmd = &cobra.Command{
	Use:   "untag <文件路径> <标签名>",
	Short: "移除文件标签",
	Long:  `移除已注册文件上的一个标签。`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		tagName := args[1]
		svc := service.NewFileService()
		if err := svc.UntagFile(filePath, tagName); err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("已从 '%s' 移除标签 '%s'\n", filePath, tagName)
	},
}

var fileInfoCmd = &cobra.Command{
	Use:   "info <文件路径>",
	Short: "查看文件详情",
	Long:  `查看已注册文件的详细信息和关联标签。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc := service.NewFileService()
		file, tags, err := svc.GetFileInfo(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("文件ID:   %d\n", file.FileID)
		fmt.Printf("路径:     %s\n", file.FilePath)
		fmt.Printf("大小:     %s\n", formatSize(file.FileSize))
		fmt.Printf("类型:     %s\n", file.FileType)
		if len(tags) > 0 {
			fmt.Println("标签:")
			for _, t := range tags {
				fmt.Printf("  [%s] %s\n", t.Category.CategoryName, t.TagName)
			}
		} else {
			fmt.Println("标签:     无")
		}
	},
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return "..." + s[len(s)-maxLen+3:]
}

var fileSearchCmd = &cobra.Command{
	Use:   "search <文件名关键词>",
	Short: "按文件名搜索",
	Long:  `在已注册文件中按文件名关键词搜索。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc := service.NewFileService()
		files, err := svc.ListFiles()
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		keyword := strings.ToLower(args[0])
		found := false
		for _, f := range files {
			if strings.Contains(strings.ToLower(f.FileName), keyword) {
				_, tags, _ := svc.GetFileInfo(f.FilePath)
				fmt.Printf("%s (%s)\n", f.FilePath, formatSize(f.FileSize))
				for _, t := range tags {
					fmt.Printf("  [%s] %s\n", t.Category.CategoryName, t.TagName)
				}
				found = true
			}
		}
		if !found {
			fmt.Println("未找到匹配的文件")
		}
	},
}

var fileExistsCmd = &cobra.Command{
	Use:   "exists <文件路径>",
	Short: "判断文件是否已索引",
	Long:  `检查指定文件是否已在数据库中注册。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc := service.NewFileService()
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

var (
	fileDiffRecursiveFlag bool
)

var fileDiffCmd = &cobra.Command{
	Use:   "diff [目录]",
	Short: "列出未索引的文件",
	Long:  `对比目录与数据库，列出存在但尚未注册的文件。默认仅扫描当前目录，-r 递归扫描。`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		svc := service.NewFileService()
		result, err := svc.DiffUnindexed(dir, fileDiffRecursiveFlag)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		if fileDiffRecursiveFlag {
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

func printTree(result *service.DiffResult) {
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

func init() {
	fileScanCmd.Flags().BoolVar(&fileScanFullFlag, "full", false, "强制全量扫描，不跳过未变更文件")
	fileDiffCmd.Flags().BoolVarP(&fileDiffRecursiveFlag, "recursive", "r", false, "递归扫描子目录，树形展示")

	FileCmd.AddCommand(fileScanCmd)
	FileCmd.AddCommand(fileDiffCmd)
	FileCmd.AddCommand(fileExistsCmd)
	FileCmd.AddCommand(fileImportCmd)
	FileCmd.AddCommand(fileListCmd)
	FileCmd.AddCommand(fileTagCmd)
	FileCmd.AddCommand(fileUntagCmd)
	FileCmd.AddCommand(fileInfoCmd)
	FileCmd.AddCommand(fileSearchCmd)
}
