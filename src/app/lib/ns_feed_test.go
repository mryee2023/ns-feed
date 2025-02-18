package lib

import (
	"fmt"
	"testing"

	"github.com/imroc/req/v3"
	"github.com/mmcdole/gofeed"
	"github.com/stretchr/testify/assert"
)

func TestLinuxDoFeed(t *testing.T) {
	feedUrl := "https://hostloc.com/forum.php?fid=45&mod=rss"
	//feedUrl = "https://rsshub.app/telegram/channel/nodeloc_rss"
	reqClient := req.C().ImpersonateChrome()
	resp, err := reqClient.R().Get(feedUrl)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	fp := gofeed.NewParser()
	feed, err := fp.ParseString(resp.String())
	assert.NoError(t, err)
	assert.NotNil(t, feed)

	for _, item := range feed.Items {
		fmt.Println(item.Title+" "+item.Link, " ", item.Published)

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
			name: "正则关键字测试",
			args: args{
				title:    "剩余价值push出港仔CMHK NAT 续费 13.88u/月",
				keywords: []string{`(?=.*港仔)(?=.*出)`, `港仔`},
			},
			want: true,
		},
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
		{
			name: "逻辑运算符关键字匹配_斯巴达_3",
			args: args{
				title:    "油管 YouTube Premium家庭组 任意区年66.99",
				keywords: []string{"youtube"},
			},
			want: true,
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
