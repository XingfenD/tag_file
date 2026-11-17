package schema

import (
	"fmt"
	"sort"
	"sync"
)

// 块注册表是进程级的：配置块的声明发生在 init 期（各消费方的 schema 文件），
// 读多写少，用一把互斥锁保证测试并发调用时也安全。内部状态与旧字段注册表
// （registry.go）互不相干：strangler 过渡期两者并存、消费方逐个切换，
// 旧注册表删除后这里成为唯一注册表。
var (
	blkMu     sync.Mutex
	blkBlocks = map[string]ConfigBlockDescriptor{}
	blkFields = map[string]*InputBoxDescriptor{}
)

// RegisterBlock 注册一个配置块，并对每个描述符跑一次 validateDescriptor 自检。
// 注册发生在 init 期，重复块 ID、重复键（跨块或块内）、描述符非法都只可能是
// 代码写错，因此直接 panic，让进程在启动时暴露问题，而不是运行时静默覆盖。
// 默认值自检（用默认值跑一次值校验）在值校验引擎就位后接入。
func RegisterBlock(b ConfigBlockDescriptor) {
	blkMu.Lock()
	defer blkMu.Unlock()

	if _, ok := blkBlocks[b.ID]; ok {
		panic(fmt.Sprintf("schema: 配置块 %q 重复注册", b.ID))
	}

	// 先整体校验再落库：任何一步失败都不留下半注册状态。
	seen := make(map[string]bool, len(b.InputBoxDescriptorList))
	for _, d := range b.InputBoxDescriptorList {
		if d == nil {
			panic(fmt.Sprintf("schema: 配置块 %q 含 nil 描述符", b.ID))
		}
		if err := validateDescriptor(*d); err != nil {
			panic(fmt.Sprintf("schema: 配置键 %q 描述符非法: %v", d.Key, err))
		}
		if seen[d.Key] {
			panic(fmt.Sprintf("schema: 配置键 %q 重复注册", d.Key))
		}
		if _, ok := blkFields[d.Key]; ok {
			panic(fmt.Sprintf("schema: 配置键 %q 重复注册", d.Key))
		}
		seen[d.Key] = true
	}

	// 注册即拷贝：调用方之后修改原值不影响注册表。
	list := make([]*InputBoxDescriptor, 0, len(b.InputBoxDescriptorList))
	for _, d := range b.InputBoxDescriptorList {
		cp := *d
		list = append(list, &cp)
		blkFields[d.Key] = &cp
	}
	blkBlocks[b.ID] = ConfigBlockDescriptor{
		ID:                     b.ID,
		Title:                  b.Title,
		Order:                  b.Order,
		InputBoxDescriptorList: list,
	}
}

// LookupInputBox 按配置键查找描述符。返回的指针归注册表所有，调用方只读。
func LookupInputBox(key string) (*InputBoxDescriptor, bool) {
	blkMu.Lock()
	defer blkMu.Unlock()

	d, ok := blkFields[key]
	return d, ok
}

// Blocks 返回所有配置块，按 Order 升序、同 Order 按 ID 升序；块内描述符按
// Key 升序。返回的是拷贝，调用方修改不影响注册表。
func Blocks() []ConfigBlockDescriptor {
	blkMu.Lock()
	defer blkMu.Unlock()

	blocks := make([]ConfigBlockDescriptor, 0, len(blkBlocks))
	for _, b := range blkBlocks {
		blocks = append(blocks, copyBlockSorted(b))
	}
	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].Order != blocks[j].Order {
			return blocks[i].Order < blocks[j].Order
		}
		return blocks[i].ID < blocks[j].ID
	})
	return blocks
}

// BlockOf 返回配置块的副本（块内描述符按 Key 升序）；未知块返回 nil。
func BlockOf(id string) *ConfigBlockDescriptor {
	blkMu.Lock()
	defer blkMu.Unlock()

	b, ok := blkBlocks[id]
	if !ok {
		return nil
	}
	cp := copyBlockSorted(b)
	return &cp
}

// AllInputBoxes 返回全部已注册描述符，跨块按 Key 升序，供 seed 使用。
// 返回的指针归注册表所有，调用方只读。
func AllInputBoxes() []*InputBoxDescriptor {
	blkMu.Lock()
	defer blkMu.Unlock()

	boxes := make([]*InputBoxDescriptor, 0, len(blkFields))
	for _, d := range blkFields {
		boxes = append(boxes, d)
	}
	sort.Slice(boxes, func(i, j int) bool {
		return boxes[i].Key < boxes[j].Key
	})
	return boxes
}

// copyBlockSorted 返回块的拷贝：描述符列表换成按 Key 升序的新切片，
// 描述符指针仍指向注册表持有的实例（与 LookupInputBox 返回的一致）。
func copyBlockSorted(b ConfigBlockDescriptor) ConfigBlockDescriptor {
	list := make([]*InputBoxDescriptor, len(b.InputBoxDescriptorList))
	copy(list, b.InputBoxDescriptorList)
	sort.Slice(list, func(i, j int) bool {
		return list[i].Key < list[j].Key
	})
	return ConfigBlockDescriptor{
		ID:                     b.ID,
		Title:                  b.Title,
		Order:                  b.Order,
		InputBoxDescriptorList: list,
	}
}
