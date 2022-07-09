package utils

import (
	"testing"
)

func TestFindWindow(t *testing.T) {
	title := "魔兽世界"
	handlers, err := FindWindowHandler(title)
	if err != nil {
		return
	}
	for _, handler := range handlers {
		PressKeyDown(uintptr(handler), uintptr(0x31))
	}
	return
}
