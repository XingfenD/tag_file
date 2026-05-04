package export

import (
	"fmt"
	"os"

	svcExport "github.com/XingfenD/tag_file/internal/service/export"
	"github.com/spf13/cobra"
)

var ExportCmd = &cobra.Command{
	Use:   "export [文件路径]",
	Short: "导出数据库为 SQL 文件",
	Long:  `将当前数据库所有表和记录导出为 SQL 文本。不指定路径则输出到标准输出。`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc := svcExport.NewService()

		var w = os.Stdout
		if len(args) > 0 {
			f, err := os.Create(args[0])
			if err != nil {
				fmt.Fprintln(os.Stderr, "错误:", err)
				os.Exit(1)
			}
			defer f.Close()
			w = f
		}

		if err := svc.ExportToSQL(w); err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}

		if len(args) > 0 {
			fmt.Printf("已导出到: %s\n", args[0])
		}
	},
}
