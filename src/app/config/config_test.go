package config

import (
	"testing"
)

func TestConfig_Storage(t *testing.T) {
	type fields struct {
		cnf *Config
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
				cnf: &Config{
					Port:              ":8080",
					TgToken:           "your_telegram_bot_token",
					NsFeed:            "https://rss.nodeseek.com",
					AdminId:           0,
					FetchTimeInterval: "10s",
					Subscribes: []*Subscribe{
						{
							Name:     "test",
							ChatId:   -989876,
							Keywords: []string{"test"},
							Status:   "on",
							Type:     "",
						},
					},
				},
			},
			args: args{
				path: "./../../etc/config.simple.yaml",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields.cnf.Storage(tt.args.path)
		})
	}
}
