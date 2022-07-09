package utils

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"keyClone/consts"
)

var (
	procEnumWindows         = user32.MustFindProc("EnumWindows")
	procGetWindowTextW      = user32.MustFindProc("GetWindowTextW")
	user32New               = windows.NewLazySystemDLL("user32.dll")
	user32                  = syscall.MustLoadDLL("user32.dll")
	procSetWindowsHookEx    = user32New.NewProc("SetWindowsHookExW")
	procLowLevelKeyboard    = user32New.NewProc("LowLevelKeyboardProc")
	procCallNextHookEx      = user32New.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx = user32New.NewProc("UnhookWindowsHookEx")
	procGetMessage          = user32New.NewProc("GetMessageW")
	procTranslateMessage    = user32New.NewProc("TranslateMessage")
	procDispatchMessage     = user32New.NewProc("DispatchMessageW")
	keyboardHook            HHOOK
	mouseHook               HHOOK
	procGetForegroundWindow = user32.MustFindProc("GetForegroundWindow")
)

type OpElement struct {
	OpType              int // 0:keyboard, 1:mouse
	KeyCode             byte
	KeyType             int // 0:down , 1:up
	CurrentWindowHandle uintptr
	MouseWheelDirection int
	MouseWParam         uintptr
	MouseLParam         uintptr
	WParam              WPARAM
}

type TagMSLLHOOKStruct struct {
	POINT       POINT
	MouseData   DWORD
	Flags       DWORD
	Time        DWORD
	DwExtraInfo uintptr
}
type MSLLHOOKSTRUCT TagMSLLHOOKStruct
type LPMSLLHOOKSTRUCT TagMSLLHOOKStruct
type PMSLLHOOKSTRUCT TagMSLLHOOKStruct

type (
	DWORD     uint32
	WPARAM    uintptr
	LPARAM    uintptr
	LRESULT   uintptr
	HANDLE    uintptr
	HINSTANCE HANDLE
	HHOOK     HANDLE
	HWND      HANDLE
)

type HOOKPROC func(int, WPARAM, LPARAM) LRESULT

type KBDLLHOOKSTRUCT struct {
	VkCode      DWORD
	ScanCode    DWORD
	Flags       DWORD
	Time        DWORD
	DwExtraInfo uintptr
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd162805.aspx
type POINT struct {
	X, Y int32
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/ms644958.aspx
type MSG struct {
	Hwnd    HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

func EnumWindows(enumFunc uintptr, lparam uintptr) (err error) {
	r1, _, e1 := syscall.Syscall(procEnumWindows.Addr(), 2, enumFunc, lparam, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func GetWindowText(hwnd syscall.Handle, str *uint16, maxCount int32) (len int32, err error) {
	r0, _, e1 := syscall.Syscall(procGetWindowTextW.Addr(), 3, uintptr(hwnd), uintptr(unsafe.Pointer(str)), uintptr(maxCount))
	len = int32(r0)
	if len == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func FindWindowHandler(title string) ([]syscall.Handle, error) {
	var hwndArr []syscall.Handle
	// var hwnd syscall.Handle
	cb := syscall.NewCallback(func(h syscall.Handle, p uintptr) uintptr {
		b := make([]uint16, 200)
		_, err := GetWindowText(h, &b[0], int32(len(b)))
		if err != nil {
			// ignore the error
			return 1 // continue enumeration
		}
		if syscall.UTF16ToString(b) == title {
			// note the window
			hwndArr = append(hwndArr, h)
			// hwnd = h
			return 1 // stop enumeration
		}
		return 1 // continue enumeration
	})
	if err := EnumWindows(cb, 0); err != nil {
		return nil, err
	}
	return hwndArr, nil
}

func SetWindowsHookEx(idHook int, lpfn HOOKPROC, hMod HINSTANCE, dwThreadId DWORD) HHOOK {
	ret, _, _ := procSetWindowsHookEx.Call(
		uintptr(idHook),
		uintptr(syscall.NewCallback(lpfn)),
		uintptr(hMod),
		uintptr(dwThreadId),
	)
	return HHOOK(ret)
}

func CallNextHookEx(hhk HHOOK, nCode int, wParam WPARAM, lParam LPARAM) LRESULT {
	ret, _, _ := procCallNextHookEx.Call(
		uintptr(hhk),
		uintptr(nCode),
		uintptr(wParam),
		uintptr(lParam),
	)
	return LRESULT(ret)
}

func UnhookWindowsHookEx(hhk HHOOK) bool {
	ret, _, _ := procUnhookWindowsHookEx.Call(
		uintptr(hhk),
	)
	return ret != 0
}

func GetMessage(msg *MSG, hwnd HWND, msgFilterMin uint32, msgFilterMax uint32) int {
	ret, _, _ := procGetMessage.Call(
		uintptr(unsafe.Pointer(msg)),
		uintptr(hwnd),
		uintptr(msgFilterMin),
		uintptr(msgFilterMax))
	return int(ret)
}

func TranslateMessage(msg *MSG) bool {
	ret, _, _ := procTranslateMessage.Call(
		uintptr(unsafe.Pointer(msg)))
	return ret != 0
}

func DispatchMessage(msg *MSG) uintptr {
	ret, _, _ := procDispatchMessage.Call(
		uintptr(unsafe.Pointer(msg)))
	return ret
}

func LowLevelKeyboardProc(nCode int, wParam WPARAM, lParam LPARAM) LRESULT {
	ret, _, _ := procLowLevelKeyboard.Call(
		uintptr(nCode),
		uintptr(wParam),
		uintptr(lParam),
	)
	return LRESULT(ret)
}

func IntInArr(a int, list []syscall.Handle) bool {
	for _, b := range list {
		if int(b) == a {
			return true
		}
	}
	return false
}

func isExclude(code int, excludeKeys []int) bool {
	for _, value := range excludeKeys {
		if code == value {
			return true
		}
	}
	return false
}

var initOpEle = OpElement{}

func KeyboardHook(channel chan *OpElement, excludeKeys []int) {
	var opEleP *OpElement = new(OpElement)
	// key.PressKeyDown(uintptr(h), consts.VK_W)
	// time.Sleep(time.Millisecond * 5000)
	// defer user32.Release()

	keyboardHook = SetWindowsHookEx(consts.WH_KEYBOARD_LL,
		func(nCode int, wparam WPARAM, lparam LPARAM) LRESULT {
			if nCode == 0 {
				opEleP = &initOpEle
				kbdStruct := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lparam))
				code := byte(kbdStruct.VkCode)            // key code
				r, _, _ := procGetForegroundWindow.Call() // current front window hwnd
				opEleP.KeyCode = code
				opEleP.CurrentWindowHandle = r
				opEleP.OpType = 0
				if (wparam == consts.WM_KEYDOWN || wparam == consts.WM_SYSKEYDOWN) && !isExclude(int(code), excludeKeys) {
					opEleP.KeyType = 0
					channel <- opEleP
				} else if (wparam == consts.WM_KEYUP || wparam == consts.WM_SYSKEYUP) && !isExclude(int(code), excludeKeys) {
					opEleP.KeyType = 1
					channel <- opEleP
				}
			}
			return CallNextHookEx(keyboardHook, nCode, wparam, lparam)
		}, 0, 0)

	var msg MSG
	for GetMessage(&msg, 0, 0, 0) != 0 {

	}
	UnhookWindowsHookEx(keyboardHook)
	keyboardHook = 0
}

func MouseHook(channel chan *OpElement) {
	var xPos int32
	var yPos int32
	var mouseLparam int32
	opMouseEleP := new(OpElement)
	mouseHook = SetWindowsHookEx(consts.WH_MOUSE_LL,
		func(nCode int, wparam WPARAM, lparam LPARAM) LRESULT {
			if nCode == 0 {
				opMouseEleP = &initOpEle
				msstruct := (*MSLLHOOKSTRUCT)(unsafe.Pointer(lparam))
				xPos = msstruct.POINT.X
				yPos = msstruct.POINT.Y
				mouseLparam = xPos | yPos<<16
				code := msstruct.MouseData
				r, _, _ := procGetForegroundWindow.Call() // current front window hwnd
				opMouseEleP.CurrentWindowHandle = r
				opMouseEleP.OpType = 1
				//fmt.Println(fmt.Sprintf("x: %d, y: %d, code: %d, wparam: %d", xPos, yPos, code, wparam))
				//if wparam == win.WM_MOUSEWHEEL {
				opMouseEleP.MouseWParam = uintptr(code)
				opMouseEleP.MouseLParam = uintptr(mouseLparam)
				opMouseEleP.WParam = wparam
				channel <- opMouseEleP
				//} else {

				//}
			}
			return CallNextHookEx(keyboardHook, nCode, wparam, lparam)
		}, 0, 0)
	var msg MSG
	for GetMessage(&msg, 0, 0, 0) != 0 {

	}

	UnhookWindowsHookEx(mouseHook)
	mouseHook = 0

}
