package lib

import (
	"fmt"
	"testing"

	"github.com/dlclark/regexp2"
	"github.com/stretchr/testify/assert"
)

func TestParseExpression(t *testing.T) {
	//examples := []string{
	//	"iPhone+频道",
	//	"苹果|安卓",
	//	"斯巴达~收",
	//}
	//for _, example := range examples {
	//	regex := ParseExpression(example)
	//	fmt.Printf("Expression: %s\nRegex: %s\n\n", example, regex)
	//}
	//
	//expr := "斯巴达~收"
	//regex := ParseExpression(expr)
	//fmt.Println("正则表达式:", regex)
	//// 测试生成的正则表达式
	//testStrings := []string{"这是斯巴达", "斯巴达 收到", "iPhone频道", "iPhone 和 频道", "这是iPhone"}
	//for _, str := range testStrings {
	//	matched, _ := regexp.MatchString(regex, str)
	//	fmt.Printf("字符串: \"%s\" 匹配结果: %v\n", str, matched)
	//}
}

func TestParseExpression1(t *testing.T) {
	type args struct {
		expr string
		text string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "兼容原有的关键字匹配",
			args: args{
				expr: "iPhone",
				text: "这是iPhone",
			},
			want: true,
		},
		{
			name: "测试一下与关系",
			args: args{
				expr: "iPhone+频道",
				text: "我喜欢听这个iPhone的信息频道",
			},
			want: true,
		},
		{
			name: "测试或关系",
			args: args{
				expr: "苹果|安卓",
				text: "我喜欢苹果手机",
			},
			want: true,
		},
		{
			name: "测试非关系",
			args: args{
				expr: "斯巴达~收",
				text: "[出]这是斯巴达",
			},
			want: true,
		},
		{
			name: "测试非关系",
			args: args{
				expr: "斯巴达~收",
				text: "[收]斯巴达小鸡一台",
			},
			want: false,
		},
		{
			name: "测试一下组合",
			args: args{
				expr: "斯巴达~收+小鸡",
				text: "",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseExpression(tt.args.expr)
			re := regexp2.MustCompile(got, 0)
			isMatch, _ := re.MatchString(tt.args.text)
			fmt.Println("正则表达式:", got, "name", tt.name)
			assert.Equal(t, tt.want, isMatch)
		})
	}
}
