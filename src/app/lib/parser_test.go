package lib

import (
	"testing"
)

func Test_match(t *testing.T) {
	type args struct {
		text string
		expr string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "验证0表达式",
			args: args{
				text: "这是一个关于iPhone的频道",
				expr: "iPhone",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "验证与表达式",
			args: args{
				text: "hello world",
				expr: "hello+world",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "验证或表达式",
			args: args{
				text: "hello world",
				expr: "hello|world",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "验证非表达式",
			args: args{
				text: "这是一个关于iPhone的频道",
				expr: "频道~iPhone",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "验证非表达式_2",
			args: args{
				text: "收一台斯巴达小鸡",
				expr: "斯巴达~收",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "验证非表达式_3",
			args: args{
				text: "出一台斯巴达小鸡",
				expr: "斯巴达~收",
			},
			want:    true,
			wantErr: false,
		},
		//{
		//	name: "验证复杂表达式",
		//	args: args{
		//		text: "这是一个关于iPhone的频道",
		//		expr: "(频道~iPhone)|(频道+关于)",
		//	},
		//	want:    true,
		//	wantErr: false,
// parseToRegex 将表达式解析为正则表达式
func parseToRegex(expr string) string {
	var regexParts []string
	orParts := strings.Split(expr, "|")

	for _, orPart := range orParts {
		andParts := strings.Split(orPart, "+")
		var andRegexParts []string

		for _, part := range andParts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "~") {
				// 处理非
				notKeyword := strings.TrimPrefix(part, "~")
				andRegexParts = append(andRegexParts, "(?!.*"+notKeyword+")")
			} else {
				andRegexParts = append(andRegexParts, "(?=.*"+part+")")
			}
		}

		regexParts = append(regexParts, strings.Join(andRegexParts, ""))
	}

	return strings.Join(regexParts, "|")
}		//},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := match(tt.args.text, tt.args.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("match() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("match() got = %v, want %v", got, tt.want)
			}
		})
	}
}
