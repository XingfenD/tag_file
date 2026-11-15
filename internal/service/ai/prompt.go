package ai

import (
	"fmt"
	"strings"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/service/tag"
)

// promptTagDepthLimit caps how deep the rendered tree goes, following the approach
// of cmd/webui/api.buildTagPath: an over-long chain of parent_id links (a cycle in
// the worst case) must truncate instead of recursing forever. ListTagTree builds its
// forest from tags without a parent, so a cycle normally drops those tags entirely
// rather than looping — the limit is insurance, not the primary guard, and that
// mirror is deliberate so both renderers stop at the same depth.
const promptTagDepthLimit = 20

// promptContract is the one output shape the model is held to, verbatim and in one
// place: ParseSuggestions decodes exactly this object, and Task 9 stores what comes
// out of it. Descriptions, confidence scores and parent tags are not part of it —
// the spec discards descriptions and never asks the model for a hierarchy.
const promptContract = `{"tags":[{"category":"<现有分类名>","tag":"<标签名>","value":"<可选,自由文本>"}]}`

const promptHead = `你是图像标签助手。请观察图片, 从下面的现有标签库里挑选贴切的标签, 每条标签属于一个分类, 可选地带一个取值。`

const promptCategories = `现有分类:`

const promptTree = `现有标签(写成「父标签 / 子标签」的完整层级, 括号里是该标签已有的预设值):`

// promptRules closes the loop between what the model is shown and what
// ParseSuggestions accepts: reuse over invention, new terms still allowed, and JSON
// with nothing but the three contract fields.
const promptRules = `输出要求:
1. 只输出 JSON: 一个 JSON 对象, 不要解释, 不要 markdown 代码围栏, 除 JSON 外不要有任何其他文字。
2. 输出格式固定为: ` + promptContract + `
3. category 必须逐字使用「现有分类」里的名字; tag 必须优先逐字使用标签路径的最后一段(例如路径「户外 / 海滩」只填「海滩」), 不要翻译、改写或加空格。
4. 该标签列出了预设值时, value 优先从预设值里选; 没有合适的预设值, value 留空字符串。
5. tags 至少包含一项, 通常 1-6 个真正贴切的标签; 确实没有完全合适的标签时给出最接近的一个, 不要返回空的 tags 数组。
6. 现有标签库里确实没有合适的标签时, 可以提出新的分类名、标签名或取值, 但不要把已有标签换个说法再说一遍。
7. 不要输出标签描述, 不要输出置信度, 不要输出父标签字段: 这三样都不参与入库。`

// BuildPrompt renders the tag library into the text half of one vision request: the
// existing categories, the tag tree as Parent / Child paths under their category
// with each tag's predefined values, and the fixed output contract. Its whole
// purpose is reuse — a model that cannot see 「海滩」 invents 「沙滩」, and the
// review queue fills up with synonyms.
//
// Everything is read once up front (all categories, the whole forest, all preset
// values) so the rendering itself makes no database call.
func (s *Service) BuildPrompt() (string, error) {
	categories, _, err := s.categorySvc.ListCategories(0, 0)
	if err != nil {
		return "", fmt.Errorf("构建提示词: 读取分类列表失败: %w", err)
	}
	nodes, err := s.tagSvc.ListTagTree("")
	if err != nil {
		return "", fmt.Errorf("构建提示词: 读取标签树失败: %w", err)
	}
	// 预设值批量一次取回: 标签树里每个名字都要用, 逐标签查库会把一次请求变成 N 次。
	// ListTagValuesBatch 按名字归组, 所以同名标签跨分类时共用同一份预设值 —— 标签名
	// 在分类内唯一, 这是既有服务层的语义, 提示词只是把它照搬出来。
	values, err := s.tagSvc.ListTagValuesBatch(collectTagNames(nodes, 0))
	if err != nil {
		return "", fmt.Errorf("构建提示词: 读取标签预设值失败: %w", err)
	}

	// ListTagTree("") 把所有分类的根标签拍平成一片森林, 这里按 CategoryID 分回去,
	// 模型才能知道每条路径属于哪个分类。分类删除前必须先清空其标签
	// (categorySvc.RemoveCategory 有此校验), 所以根标签的分类名必然能在上面清单里找到。
	rootsByCategory := make(map[uint][]tag.TagTreeNode, len(categories))
	for i := range nodes {
		categoryID := nodes[i].Tag.CategoryID
		rootsByCategory[categoryID] = append(rootsByCategory[categoryID], nodes[i])
	}

	var b strings.Builder
	b.WriteString(promptHead)

	b.WriteString("\n\n" + promptCategories + "\n")
	if len(categories) == 0 {
		b.WriteString("- (暂无分类, 可自由提出新的分类名)\n")
	}
	for _, category := range categories {
		fmt.Fprintf(&b, "- %s\n", category.CategoryName)
	}

	b.WriteString("\n" + promptTree + "\n")
	rendered := false
	for _, category := range categories {
		roots := rootsByCategory[category.CategoryID]
		if len(roots) == 0 {
			// 空分类只出现在分类清单里, 在标签树一节不重复列, 免得看起来像有个
			// 名字叫它的路径。
			continue
		}
		rendered = true
		fmt.Fprintf(&b, "%s\n", category.CategoryName)
		writeTagNodes(&b, roots, "", 0, values)
	}
	if !rendered {
		b.WriteString("- (暂无标签, 可自由提出新的标签名)\n")
	}

	fmt.Fprintf(&b, "\n%s\n", promptRules)
	return b.String(), nil
}

// writeTagNodes renders one forest level: each node on its own line as the full
// path from its category root, indented by depth so the hierarchy is visible even
// though the path already spells it out.
func writeTagNodes(b *strings.Builder, nodes []tag.TagTreeNode, path string, depth int, values map[string][]model.TagValue) {
	// 先判空: 一个叶子标签也会带着一片空的 Children 递归进来, 若先报深度就变成
	// 「刚好停在上限的标签库」谎称自己被截断了。
	if len(nodes) == 0 {
		return
	}
	if depth > promptTagDepthLimit {
		fmt.Fprintf(b, "%s- …(标签层级超过 %d 层, 已截断)\n", strings.Repeat("  ", depth+1), promptTagDepthLimit)
		return
	}
	indent := strings.Repeat("  ", depth+1)
	for i := range nodes {
		name := nodes[i].Tag.TagName
		nodePath := name
		if path != "" {
			nodePath = path + " / " + name
		}
		fmt.Fprintf(b, "%s- %s", indent, nodePath)
		if preset := presetValueText(values[name]); preset != "" {
			fmt.Fprintf(b, "(预设值: %s)", preset)
		}
		b.WriteString("\n")
		writeTagNodes(b, nodes[i].Children, nodePath, depth+1, values)
	}
}

// collectTagNames walks the forest for the names ListTagValuesBatch needs, with the
// same depth limit as the renderer so the two agree on which tags count as shown.
func collectTagNames(nodes []tag.TagTreeNode, depth int) []string {
	if depth > promptTagDepthLimit {
		return nil
	}
	var names []string
	for i := range nodes {
		names = append(names, nodes[i].Tag.TagName)
		names = append(names, collectTagNames(nodes[i].Children, depth+1)...)
	}
	return names
}

// presetValueText joins a tag's predefined values for the prompt.
func presetValueText(tagValues []model.TagValue) string {
	texts := make([]string, 0, len(tagValues))
	for _, value := range tagValues {
		if value.Value == "" {
			continue
		}
		texts = append(texts, value.Value)
	}
	return strings.Join(texts, "、")
}
