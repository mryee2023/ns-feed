package lib

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mmcdole/gofeed"
)

func TestLinuxDoFeed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	fp := gofeed.NewParser()
	feed, err := fp.ParseURLWithContext("https://linux.do/latest.rss", ctx)
	if err != nil {
		fmt.Println(err)
	}
	if feed == nil {
		fmt.Println("feed is nil")
		return
	}

	for _, item := range feed.Items {
		fmt.Println(item.Title, " ", item.PublishedParsed.Add(time.Hour*8).Format("2006-01-02 15:04:05"), " ", item.Link)
	}
}

func Test_hasKeyword(t *testing.T) {
	type args struct {
		title    string
		keywords []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "存量关键字匹配",
			args: args{
				title:    "剩余价值➕push出港仔CMHK NAT 续费 13.88u/月",
				keywords: []string{"bgp", "探针", "bgp.gd", "港仔", "mk", "boil", "zgo", "lala", "bage"},
			},
			want: true,
		},
		{
			name: "逻辑运算符关键字匹配",
			args: args{
				title:    "剩余价值➕push出港仔CMHK NAT 续费 13.88u/月",
				keywords: []string{"bgp", "探针", "bgp.gd", "港仔~NAT", "mk", "boil", "zgo", "lala", "bage"},
			},
			want: false,
		},
		{
			name: "逻辑运算符关键字匹配_斯巴达_1",
			args: args{
				title:    "[收]斯巴达小鸡一个",
				keywords: []string{"斯巴达"},
			},
			want: true,
		},
		{
			name: "逻辑运算符关键字匹配_斯巴达_2",
			args: args{
				title:    "[收]斯巴达小鸡一个",
				keywords: []string{"斯巴达~收"},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasKeyword(tt.args.title, tt.args.keywords); got != tt.want {
				t.Errorf("hasKeyword() = %v, want %v", got, tt.want)
			}
		})
	}
}
