package file

import (
	"log"

	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

// RepairFileTypes 是 0.14.1 类型判定的存量半边：在那之前扫入的行把扩展名
// 按字面存进了 file_type（musl 容器里 mime.TypeByExtension 查空的产物），而
// 扫描只建不改、重扫跳过已索引路径，升级后这批文件的媒体分类与可播判定照样
// 失配。启动时对所有「非 MIME 形态」（类型串里没有 /）的行现算 fileTypeByExt，
// 值变了才写回。幂等靠值检：兜底表外的裸扩展名（.md 之类）重算得自身，第二
// 遍起零写回；无扩展名或空类型同理由 want==stored/空串短路。
func RepairFileTypes(db *gorm.DB) error {
	var files []model.File
	if err := db.Where("file_type NOT LIKE ?", "%/%").Find(&files).Error; err != nil {
		return err
	}
	fixed := 0
	for i := range files {
		want := fileTypeByExt(files[i].FilePath)
		if want == "" || want == files[i].FileType {
			continue
		}
		if err := db.Model(&files[i]).Update("file_type", want).Error; err != nil {
			return err
		}
		fixed++
	}
	if fixed > 0 {
		log.Printf("[file] repaired %d stored file type(s) to MIME form", fixed)
	}
	return nil
}
