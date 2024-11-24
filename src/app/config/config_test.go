package config

import (
	"testing"
)

func TestConfig_Storage(t *testing.T) {
	type fields struct {
		TgToken     string
		TgChatId    int64
		NsFeed      string
		AlterChatId int64
		Keywords    []string
	}
	type args struct {
		path string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "TestConfig_Storage",
			fields: fields{
				TgToken:     "myToken",
				TgChatId:    -123,
				NsFeed:      "https://rss.nodeseek.com",
				AlterChatId: -456,
				Keywords:    []string{"keyword1", "keyword2"},
			},
			args: args{
				path: "./../../etc/config.simple.yaml",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				TgToken:     tt.fields.TgToken,
				TgChatId:    tt.fields.TgChatId,
				NsFeed:      tt.fields.NsFeed,
				AlterChatId: tt.fields.AlterChatId,
				Keywords:    tt.fields.Keywords,
			}
			c.Storage(tt.args.path)
		})
	}
}
