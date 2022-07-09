package main

import (
	"fmt"
	"github.com/lxn/win"
	"keyClone/utils"
	"log"
)

func executeOp(channel chan *utils.OpElement) {
	fmt.Println("KeyClone process startted")
	lastKeyOp := ""
	//var lastKeyOp string
	const title = "魔兽世界" // set the window text title
	//var currentForceHandle uintptr
	//leftButtonPressed := false
	//rightButtonPressed := false

	hArr, err := utils.FindWindowHandler(title)
	if err != nil {
		log.Fatal(err)
	}
	for opElement := range channel {
		currentForceHandle := opElement.CurrentWindowHandle
		if !utils.IntInArr(int(currentForceHandle), hArr) {
			continue
		}
		if opElement.OpType == 0 { // keyPress
			for _, value := range hArr {
				value := value
				go func() {
					if uintptr(value) == currentForceHandle { // current windows's keyStroke
						return
					}
					if opElement.KeyType == 0 && (string(opElement.KeyCode)+"DOWN") != lastKeyOp { // key press down and different with the last key stroke
						lastKeyOp = string(opElement.KeyCode) + "DOWN"
						utils.PressKeyDown(uintptr(value), uintptr(opElement.KeyCode))
					} else if opElement.KeyType == 1 && (string(opElement.KeyCode)+"UP") != lastKeyOp { // key press up and different with the last key stroke
						lastKeyOp = string(opElement.KeyCode) + "UP"
						utils.PressKeyUp(uintptr(value), uintptr(opElement.KeyCode))
					}
				}()
			}
		} else { // mouse wheel
			for _, value := range hArr {
				value := value
				//if opElement.WParam == win.WM_LBUTTONDOWN {
				//	leftButtonPressed = true
				//}
				//if opElement.WParam == win.WM_LBUTTONUP {
				//	leftButtonPressed = false
				//}
				//if opElement.WParam == win.WM_RBUTTONDOWN {
				//	rightButtonPressed = true
				//}
				//if opElement.WParam == win.WM_RBUTTONUP {
				//	rightButtonPressed = false
				//}
				go func() {
					if uintptr(value) == currentForceHandle {
						return
					}
					wParam := opElement.MouseWParam
					if opElement.WParam == win.WM_RBUTTONDOWN || opElement.WParam == win.WM_RBUTTONUP {
						return
					}
					//if leftButtonPressed && opElement.WParam == win.WM_MOUSEMOVE {
					//	wParam = win.MK_LBUTTON
					//}
					//if rightButtonPressed && opElement.WParam == win.WM_MOUSEMOVE {
					//	wParam = win.MK_RBUTTON
					//}
					utils.PostMessage(uintptr(value), uint32(opElement.WParam), wParam, opElement.MouseLParam) // post mouse wheel, see also https://docs.microsoft.com/en-us/windows/win32/inputdev/wm-mousewheel
				}()
			}
		}
	}
}

func main() {
	var channel = make(chan *utils.OpElement, 1000)
	go executeOp(channel)
	go utils.MouseHook(channel)
	utils.KeyboardHook(channel, []int{65, 68, 69, 81, 83, 87, 27, 13, 77, 144})
}
