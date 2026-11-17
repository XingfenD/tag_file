package schema

import "fmt"

// ValidateInputBox 按描述符校验存储形态的取值（spec §5.2 管线）：
// 描述符合法性（与取值无关，先于值校验）→ 容器解析（Single 原样；
// Array/DynamicArray 必须是 JSON 字符串数组）→ 容器级指令（收到原始
// value）→ 值级/项级指令（Single 跑一组；Array 逐位；DynamicArray 每项
// 跑同一组）。错误文案为中文、含合法范围，不含键名——由调用方包装键名；
// 项级失败拼「第 %d 项%s」（下标从 1）。只跑 RoleValidate（有 Check），
// render/behavior 指令跳过。
func ValidateInputBox(d InputBoxDescriptor, value string) error {
	if err := validateDescriptor(d); err != nil {
		return err
	}
	rs := RulesOf(d)
	if err := checkRuleKindsKnown(rs); err != nil {
		return err
	}

	if d.Container == ContainerSingle {
		return runRules(rs.Rules, value)
	}

	items, err := parseStringArray(value)
	if err != nil {
		return err
	}
	for _, r := range rs.ContainerRules {
		if err := runRule(r, value); err != nil {
			return err
		}
	}
	switch d.Container {
	case ContainerDynamicArray:
		for i, item := range items {
			if err := runRules(rs.Rules, item); err != nil {
				return fmt.Errorf("第 %d 项%s", i+1, err)
			}
		}
	case ContainerArray:
		// fixed_arity 已保证长度与逐位指令组一致；越界防御性跳过。
		for i, item := range items {
			if i >= len(rs.PositionalRules) {
				break
			}
			if err := runRules(rs.PositionalRules[i], item); err != nil {
				return fmt.Errorf("第 %d 项%s", i+1, err)
			}
		}
	}
	return nil
}

// runRules 逐条执行指令组，返回第一条失败（AND 语义，无短路组合逻辑）。
func runRules(rules []Rule, value string) error {
	for _, r := range rules {
		if err := runRule(r, value); err != nil {
			return err
		}
	}
	return nil
}

// runRule 执行单条指令：只跑 RoleValidate；render/behavior 无 Check，跳过。
// 未知 kind 与缺失 Check 已由 checkRuleKindsKnown 兜底，此处防御性放行。
func runRule(r Rule, value string) error {
	if r.Role != RoleValidate {
		return nil
	}
	def, ok := ruleDefs[r.Kind]
	if !ok || def.Check == nil {
		return nil
	}
	return def.Check(r.Params, value)
}
