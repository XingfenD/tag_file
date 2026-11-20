package ai

import (
	"fmt"
	"strings"

	"github.com/XingfenD/tag_file/internal/model"
)

// visionFreePrompt is the vision half of the two-pass refine pipeline: the model
// just describes what it sees, and the format discipline moves to BuildRefinePrompt.
// Deliberately no contract and no JSON wording — an answer forced into free text is
// what gets stored per frame, and a model shown the contract cannot ignore it.
const visionFreePrompt = `你是图像标签助手。观察图片, 描述画面内容并列出你看到的标签, 自由文本即可, 不需要任何格式。`

// promptRefineHead frames the second pass: nothing is observed here, the model only
// tidies text it is given, which is what rule 6 then holds it to.
const promptRefineHead = `你是标签整理助手。下面是同一段视频逐帧的原始观察记录和现有标签库: 请只依据这些文字, 把它们整理成一份归一、去重后的标签列表。`

const promptRefineFrames = `各帧原始观察记录:`

// promptRefineRules is the refine counterpart of promptRules: same fixed contract,
// but aimed at merging several raw observations instead of looking at a picture —
// verbatim library names with synonyms normalised (「沙滩」→「海滩」 is the exact
// failure the library rendering exists to prevent), one entry per meaning across
// frames, preset values first, and invention limited to re-wording what is written.
const promptRefineRules = `整理要求:
1. 只输出 JSON: 一个 JSON 对象, 不要解释, 不要 markdown 代码围栏, 除 JSON 外不要有任何其他文字。
2. 输出格式固定为: ` + promptContract + `
3. category 与 tag 必须逐字使用上面标签库里的名字; 观察记录里的同义词要归一到库内已有标签, 例如把「沙滩」归一为「海滩」。
4. 不同帧说到同一个含义时, 合并成一条, 不要为同一个含义输出多条记录。
5. 该标签列出了预设值时, value 优先从预设值里选; 没有合适的预设值, value 留空字符串。
6. 只能依据上面的原始观察记录整理, 不得凭空添加记录里没有的内容: 你看不到画面, 也不要臆想画面。
7. 库里确实没有合适标签时, 可以提出新的分类名、标签名或取值, 但不要换个说法重复已有的标签。`

// BuildRefinePrompt assembles the text-only second pass of the refine pipeline:
// the same tag library section the vision prompt showed (so normalisation has
// something to normalise to), each frame's stored raw answer under a
// 「第 %d 帧:」 label carrying its verbatim FrameIndex, and the refine rules.
// It reads no frames and calls no model — Task 6 hands it what the frame
// repository kept.
func (s *Service) BuildRefinePrompt(frames []model.AIFrameOutput) (string, error) {
	library, err := s.renderPromptLibrary()
	if err != nil {
		return "", fmt.Errorf("构建整理提示词: %w", err)
	}

	var b strings.Builder
	b.WriteString(promptRefineHead)
	b.WriteString("\n\n")
	b.WriteString(library)
	fmt.Fprintf(&b, "\n%s\n", promptRefineFrames)
	for _, frame := range frames {
		// FrameIndex 用原值逐帧标注, RawText 逐字附上: 整理只对着这份文字负责。
		fmt.Fprintf(&b, "第 %d 帧:\n%s\n", frame.FrameIndex, frame.RawText)
	}
	fmt.Fprintf(&b, "\n%s\n", promptRefineRules)
	return b.String(), nil
}
