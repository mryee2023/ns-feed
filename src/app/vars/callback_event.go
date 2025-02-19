package vars

import (
	"encoding/json"
)

type Event string

const (
	EventSelectFeed    Event = "01"
	EventBackToMain    Event = "02"
	EventDeleteKeyword Event = "03"
	EventAddKeyword    Event = "04"
	EventConfirmDelete Event = "05"
	EventOn            Event = "06"
	EventOff           Event = "07"
)

type CallbackEvent[T CallbackData] struct {
	Event string `json:"e"`
	Data  T      `json:"d,omitempty"`
}

func (c *CallbackEvent[T]) Param() string {
	c.Event = c.Data.Method()
	b, _ := json.Marshal(c)
	return string(b)
}

type CallbackData interface {
	Method() string
}

// CallbackFeedData 简化版的Feed数据结构
type CallbackFeedData struct {
	FeedId string `json:"i"`
}

func (f CallbackFeedData) Method() string {
	return string(EventSelectFeed)
}

type CallbackBackToMain struct {
}

func (b CallbackBackToMain) Method() string {
	return string(EventBackToMain)
}

type CallbackDeleteKeyword struct {
	Keyword string `json:"k"`
	FeedId  string `json:"i"`
}

func (c CallbackDeleteKeyword) Method() string {
	return string(EventDeleteKeyword)
}

type CallbackAddKeyword struct {
	FeedId string `json:"i"`
}

func (c CallbackAddKeyword) Method() string {
	return string(EventAddKeyword)
}

// CallbackConfirmDelete 确认删除的回调数据结构
type CallbackConfirmDelete struct {
	Keyword string `json:"k"`
	FeedId  string `json:"i"`
}

func (c CallbackConfirmDelete) Method() string {
	return string(EventConfirmDelete)
}

type CallbackStatusOn struct {
	ChatId int64 `json:"c"`
}

func (c CallbackStatusOn) Method() string {
	return string(EventOn)
}

type CallbackStatusOff struct {
	ChatId int64 `json:"c"`
}

func (c CallbackStatusOff) Method() string {
	return string(EventOff)
}
