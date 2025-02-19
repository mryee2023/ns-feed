package vars

import (
	"fmt"
	"testing"
)

func TestCallbackEvent_Param(t *testing.T) {
	type testCase[T CallbackData] struct {
		name string
		c    CallbackEvent[T]
	}
	tests := []testCase[CallbackDeleteKeyword]{
		{
			name: "Test CallbackEvent.Param",
			c: CallbackEvent[CallbackDeleteKeyword]{
				Event: "03",
				Data: CallbackDeleteKeyword{
					Keyword: `(?=.*(港仔|ggy|claw).*)`,
					FeedId:  "ns",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.c.Param()
			fmt.Println(v)
			fmt.Println(len(v))
		})
	}
}
