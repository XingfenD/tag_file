package schema

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// Validate 按字段类型校验存储形态的值。错误文案为中文、含合法范围，不含键名——
// 由调用方包装键名。未知类型返回错误，绝不静默放行。
func Validate(f Field, value string) error {
	switch f.Type {
	case TypeString:
		return validateString(f, value)
	case TypeText, TypeSecret:
		return nil
	case TypeURL:
		return validateURL(value)
	case TypeInt:
		return validateInt(f, value)
	case TypeFloat:
		return validateFloat(f, value)
	case TypeDuration:
		return validateDuration(f, value)
	case TypeBool:
		return validateBool(value)
	case TypeEnum:
		return validateEnum(f, value)
	case TypeList:
		return validateList(f, value)
	case TypeRegex:
		return fmt.Errorf("正则类型只能作为列表项类型使用")
	default:
		return fmt.Errorf("未知的配置类型 '%s'，无法校验", f.Type)
	}
}

func validateString(f Field, value string) error {
	if f.Pattern == "" {
		return nil
	}
	re, err := regexp.Compile(f.Pattern)
	if err != nil {
		return fmt.Errorf("匹配模式无效: %v", err)
	}
	if !re.MatchString(value) {
		return fmt.Errorf("取值不符合匹配模式 %s", f.Pattern)
	}
	return nil
}

func validateURL(value string) error {
	if value == "" {
		return fmt.Errorf("取值必须是 http 或 https 地址，不能为空")
	}
	u, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("取值必须是合法的 http 或 https 地址")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("取值必须以 http:// 或 https:// 开头")
	}
	if u.Host == "" {
		return fmt.Errorf("取值必须包含主机名")
	}
	return nil
}

func validateInt(f Field, value string) error {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("取值必须是整数")
	}
	return checkBounds(f, float64(n))
}

func validateFloat(f Field, value string) error {
	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("取值必须是数字")
	}
	return checkBounds(f, n)
}

func validateDuration(f Field, value string) error {
	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("取值必须是秒数")
	}
	return checkBounds(f, n)
}

func validateBool(value string) error {
	if value != "true" && value != "false" {
		return fmt.Errorf("取值只能是 true 或 false")
	}
	return nil
}

func validateEnum(f Field, value string) error {
	options := make([]string, 0, len(f.Options))
	for _, o := range f.Options {
		if o.Value == value {
			return nil
		}
		options = append(options, o.Value)
	}
	if len(options) == 0 {
		return fmt.Errorf("没有可选的取值")
	}
	return fmt.Errorf("取值必须是以下之一: %s", strings.Join(options, "、"))
}

func validateList(f Field, value string) error {
	if !strings.HasPrefix(strings.TrimSpace(value), "[") {
		return fmt.Errorf("取值必须是 JSON 字符串数组，例如 [\"a\",\"b\"]")
	}
	var items []string
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return fmt.Errorf("取值必须是 JSON 字符串数组，例如 [\"a\",\"b\"]")
	}
	switch f.ItemType {
	case "", TypeString:
		return nil
	case TypeRegex:
		for i, item := range items {
			if _, err := regexp.Compile(item); err != nil {
				return fmt.Errorf("第 %d 项不是合法的正则表达式: %v", i+1, err)
			}
		}
		return nil
	default:
		return fmt.Errorf("未知的列表项类型 '%s'，无法校验", f.ItemType)
	}
}

func checkBounds(f Field, v float64) error {
	if f.Min != nil {
		if f.MinExclusive {
			if v <= *f.Min {
				return boundsError(f)
			}
		} else if v < *f.Min {
			return boundsError(f)
		}
	}
	if f.Max != nil {
		if f.MaxExclusive {
			if v >= *f.Max {
				return boundsError(f)
			}
		} else if v > *f.Max {
			return boundsError(f)
		}
	}
	return nil
}

func boundsError(f Field) error {
	return fmt.Errorf("取值超出合法范围，应为 %s", describeRange(f))
}

// describeRange 用区间写法描述合法范围，排他边界用圆括号表示。
func describeRange(f Field) string {
	left, right := "[", "]"
	if f.MinExclusive || f.Min == nil {
		left = "("
	}
	if f.MaxExclusive || f.Max == nil {
		right = ")"
	}
	switch {
	case f.Min != nil && f.Max != nil:
		return left + formatNum(*f.Min) + ", " + formatNum(*f.Max) + right
	case f.Min != nil:
		return left + formatNum(*f.Min) + ", +∞" + right
	case f.Max != nil:
		return "(-∞, " + formatNum(*f.Max) + right
	default:
		return "任意值"
	}
}

func formatNum(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
