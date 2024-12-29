package lib

import (
	"fmt"
	"regexp"
	"strings"
)

// escapeString 处理字符串中的特殊字符，使其适用于正则表达式。
func escapeString(s string) string {
	return regexp.QuoteMeta(s)
}

// ParseExpression 解析自定义语法的输入表达式并转换为正则表达式。
func ParseExpression(expr string) string {
	orParts := strings.Split(expr, "|")
	var orRegexParts []string
	for _, orPart := range orParts {
		conjunctionParts := strings.Split(orPart, "+")
		var conjunctionRegexParts []string
		// 用于存放排除的项
		var negationParts []string
		// 用于存放需要包含的项
		var inclusionParts []string
		for _, conjPart := range conjunctionParts {
			conjPart = strings.TrimSpace(conjPart)

			negationPartsArray := strings.Split(conjPart, "~")
			for i, s := range negationPartsArray {
				if i == 0 {
					inclusionParts = append(inclusionParts, escapeString(s))
					continue
				}
				negationParts = append(negationParts, escapeString(negationPartsArray[i]))
			}

			//if strings.HasPrefix(conjPart, "~") {
			//	// 处理非关系，记录需排除的项
			//	negationParts = append(negationParts, escapeString(conjPart[1:]))
			//} else {
			//	// 记录需包含的项
			//	inclusionParts = append(inclusionParts, escapeString(conjPart))
			//}
		}
		// 构建包含的正则部分
		includeRegex := ""
		for _, part := range inclusionParts {
			includeRegex += fmt.Sprintf("(?=.*%s)", part)
		}
		// 构建排除的正则部分
		excludeRegex := ""
		for _, part := range negationParts {
			excludeRegex += fmt.Sprintf("(?!.*%s)", part)
		}
		// 合并包含和排除的正则部分
		finalRegex := fmt.Sprintf("^%s%s.*$", includeRegex, excludeRegex)
		conjunctionRegexParts = append(conjunctionRegexParts, finalRegex)
		orRegexParts = append(orRegexParts, finalRegex)
	}

	finalRegex := fmt.Sprintf("(%s)", strings.Join(orRegexParts, "|"))
	return finalRegex
}
