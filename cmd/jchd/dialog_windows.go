package main

import (
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")

	procDialogBoxIndirectParamW       = user32.NewProc("DialogBoxIndirectParamW")
	procEndDialog                     = user32.NewProc("EndDialog")
	procSendDlgItemMessageW           = user32.NewProc("SendDlgItemMessageW")
	procSetDlgItemTextW               = user32.NewProc("SetDlgItemTextW")
	procGetDlgItemTextW               = user32.NewProc("GetDlgItemTextW")
	procSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	procGetModuleHandleW              = kernel32.NewProc("GetModuleHandleW")
	procCreateActCtxW                 = kernel32.NewProc("CreateActCtxW")
	procActivateActCtx                = kernel32.NewProc("ActivateActCtx")
	procSHBrowseForFolderW            = shell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDListW          = shell32.NewProc("SHGetPathFromIDListW")
	procCoTaskMemFree                 = windows.NewLazySystemDLL("ole32.dll").NewProc("CoTaskMemFree")
)

// Window styles, messages and control classes used by the dialog.
const (
	wsChild   = 0x40000000
	wsVisible = 0x10000000
	wsPopup   = 0x80000000
	wsCaption = 0x00C00000
	wsSysMenu = 0x00080000
	wsTabStop = 0x00010000
	wsVScroll = 0x00200000
	wsBorder  = 0x00800000

	dsSetFont    = 0x40
	dsModalFrame = 0x80
	dsCenter     = 0x800

	bsDefPushButton = 0x1
	bsAutoCheckBox  = 0x3
	cbsDropDownList = 0x3
	esAutoHScroll   = 0x80

	classButton = 0x80
	classEdit   = 0x81
	classStatic = 0x82
	classCombo  = 0x85

	wmInitDialog = 0x110
	wmCommand    = 0x111
	bmGetCheck   = 0xF0
	bmSetCheck   = 0xF1
	cbAddString  = 0x143
	cbGetCurSel  = 0x147
	cbSetCurSel  = 0x14E
	enChange     = 0x300

	idOK      = 1
	idCancel  = 2
	idSound   = 101
	idFilter  = 102
	idScreens = 103
	idFolder  = 104
	idBrowse  = 105
	idStatus  = 106
)

// The choices in the dialog's lists, in order.
var (
	filterNames  = []string{"sharp", "nearest", "smooth"}
	filterLabels = []string{"Sharp (recommended)", "Blocky pixels", "Smooth"}
	screenLabels = []string{"Johnny on every monitor", "Johnny on the main monitor only"}
)

// dialogTemplate builds a dialog in memory, as a resource compiler would
// from a .rc file. Positions and sizes are in dialog units, which follow
// the dialog's font.
type dialogTemplate struct {
	w []uint16
}

func (t *dialogTemplate) dword(v uint32) { t.w = append(t.w, uint16(v), uint16(v>>16)) }
func (t *dialogTemplate) word(v uint16)  { t.w = append(t.w, v) }

func (t *dialogTemplate) text(s string) {
	t.w = append(t.w, utf16.Encode([]rune(s))...)
	t.w = append(t.w, 0)
}

func (t *dialogTemplate) rect(x, y, w, h int16) {
	t.w = append(t.w, uint16(x), uint16(y), uint16(w), uint16(h))
}

func newDialogTemplate(title string, w, h int16) *dialogTemplate {
	t := &dialogTemplate{}
	t.dword(wsPopup | wsCaption | wsSysMenu | dsModalFrame | dsCenter | dsSetFont)
	t.dword(0) // extended style
	t.word(0)  // number of controls, counted in add
	t.rect(0, 0, w, h)
	t.word(0) // no menu
	t.word(0) // standard dialog class
	t.text(title)
	t.word(9) // font size in points
	t.text("Segoe UI")
	return t
}

// add adds a control of a standard class.
func (t *dialogTemplate) add(class uint16, style uint32, x, y, w, h int16, id uint16, text string) {
	if len(t.w)%2 == 1 {
		t.word(0) // controls start on a 4-byte boundary
	}
	t.dword(wsChild | wsVisible | style)
	t.dword(0)
	t.rect(x, y, w, h)
	t.word(id)
	t.word(0xFFFF)
	t.word(class)
	t.text(text)
	t.word(0) // no creation data
	t.w[4]++
}

// settingsDialog lays out the settings dialog.
func settingsDialog() *dialogTemplate {
	t := newDialogTemplate("Johnny Castaway HD settings", 250, 150)
	t.add(classButton, bsAutoCheckBox|wsTabStop, 8, 8, 150, 10, idSound, "Play sounds")
	t.add(classStatic, 0, 8, 28, 60, 8, 0, "Picture:")
	t.add(classCombo, cbsDropDownList|wsVScroll|wsTabStop, 72, 26, 170, 80, idFilter, "")
	t.add(classStatic, 0, 8, 46, 60, 8, 0, "Monitors:")
	t.add(classCombo, cbsDropDownList|wsVScroll|wsTabStop, 72, 44, 170, 60, idScreens, "")
	t.add(classStatic, 0, 8, 66, 234, 10, 0, "Folder with the game files (leave empty to look next to the screensaver):")
	t.add(classEdit, esAutoHScroll|wsBorder|wsTabStop, 8, 78, 180, 14, idFolder, "")
	t.add(classButton, wsTabStop, 192, 78, 50, 14, idBrowse, "Browse...")
	t.add(classStatic, 0, 8, 98, 234, 26, idStatus, "")
	t.add(classButton, bsDefPushButton|wsTabStop, 138, 128, 50, 14, idOK, "OK")
	t.add(classButton, wsTabStop, 192, 128, 50, 14, idCancel, "Cancel")
	return t
}

// dataStatus tells whether the game files are found in dir, or where the
// screensaver looks for them if dir is empty.
func dataStatus(dir string) string {
	if dir == "" {
		if d, err := defaultDataDir(); err == nil {
			return "Game files found in " + d + "."
		}
		return "Game files not found next to the screensaver. Choose the folder with RESOURCE.MAP, RESOURCE.001 and SCRANTIC.SCR from the original."
	}
	if _, ok := findDataDir([]string{dir}); ok {
		return "Game files found."
	}
	return "This folder doesn't have the game files: RESOURCE.MAP, RESOURCE.001 and SCRANTIC.SCR from the original."
}

// showSettings shows the settings dialog over the window parent (0: none)
// and saves the settings when OK is pressed.
func showSettings(parent uintptr) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED) // for the folder browser
	const perMonitorAwareV2 = ^uintptr(3)                       // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2
	procSetProcessDpiAwarenessContext.Call(perMonitorAwareV2)
	useVisualStyles()

	s := loadSettings()
	send := func(h, id uintptr, msg, wp, lp uintptr) uintptr {
		r, _, _ := procSendDlgItemMessageW.Call(h, id, msg, wp, lp)
		return r
	}
	setText := func(h, id uintptr, text string) {
		p, _ := windows.UTF16PtrFromString(text)
		procSetDlgItemTextW.Call(h, id, uintptr(unsafe.Pointer(p)))
	}
	getText := func(h, id uintptr) string {
		buf := make([]uint16, 1024)
		procGetDlgItemTextW.Call(h, id, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		return strings.TrimSpace(windows.UTF16ToString(buf))
	}
	addStrings := func(h, id uintptr, labels []string, selected int) {
		for _, l := range labels {
			p, _ := windows.UTF16PtrFromString(l)
			send(h, id, cbAddString, 0, uintptr(unsafe.Pointer(p)))
		}
		send(h, id, cbSetCurSel, uintptr(selected), 0)
	}

	proc := func(h, msg, wp, lp uintptr) uintptr {
		switch msg {
		case wmInitDialog:
			if !s.Mute {
				send(h, idSound, bmSetCheck, 1, 0)
			}
			filter := 0
			for i, name := range filterNames {
				if name == s.Filter {
					filter = i
				}
			}
			addStrings(h, idFilter, filterLabels, filter)
			screens := 0
			if s.MainOnly {
				screens = 1
			}
			addStrings(h, idScreens, screenLabels, screens)
			setText(h, idFolder, s.DataDir)
			setText(h, idStatus, dataStatus(s.DataDir))
			return 1
		case wmCommand:
			switch id, code := wp&0xFFFF, wp>>16&0xFFFF; {
			case id == idFolder && code == enChange:
				setText(h, idStatus, dataStatus(getText(h, idFolder)))
			case id == idBrowse:
				if dir, ok := browseFolder(h); ok {
					setText(h, idFolder, dir)
				}
			case id == idOK:
				s.Mute = send(h, idSound, bmGetCheck, 0, 0) == 0
				s.Filter = filterNames[min(int(send(h, idFilter, cbGetCurSel, 0, 0)), len(filterNames)-1)]
				s.MainOnly = send(h, idScreens, cbGetCurSel, 0, 0) == 1
				s.DataDir = getText(h, idFolder)
				if err := saveSettings(s); err != nil {
					message(h, "The settings could not be saved:\n"+err.Error())
					return 1
				}
				procEndDialog.Call(h, idOK)
			case id == idCancel:
				procEndDialog.Call(h, idCancel)
			}
			return 1
		}
		return 0
	}

	t := settingsDialog()
	module, _, _ := procGetModuleHandleW.Call(0)
	r, _, err := procDialogBoxIndirectParamW.Call(module, uintptr(unsafe.Pointer(&t.w[0])), parent, windows.NewCallback(proc), 0)
	runtime.KeepAlive(t)
	if int(r) == -1 || r == 0 && err != windows.ERROR_SUCCESS {
		log.Printf("settings dialog: %v", err)
	}
}

// useVisualStyles makes the dialog's controls look as in the rest of
// Windows rather than as in Windows 95. That takes version 6 of the common
// controls, which a program asks for in its manifest; without one, Windows'
// own manifest in shell32.dll (resource 124) does the same.
func useVisualStyles() {
	sys, err := windows.GetSystemDirectory()
	if err != nil {
		return
	}
	source, _ := windows.UTF16PtrFromString(filepath.Join(sys, "shell32.dll"))
	ctx := struct {
		size         uint32
		flags        uint32
		source       *uint16
		arch, lang   uint16
		assemblyDir  *uint16
		resourceName uintptr
		appName      *uint16
		module       uintptr
	}{flags: 0x8, source: source, resourceName: 124} // ACTCTX_FLAG_RESOURCE_NAME_VALID
	ctx.size = uint32(unsafe.Sizeof(ctx))
	h, _, _ := procCreateActCtxW.Call(uintptr(unsafe.Pointer(&ctx)))
	if windows.Handle(h) == windows.InvalidHandle {
		return
	}
	var cookie uintptr
	procActivateActCtx.Call(h, uintptr(unsafe.Pointer(&cookie)))
}

// browseFolder lets the user choose a folder.
func browseFolder(owner uintptr) (string, bool) {
	title, _ := windows.UTF16PtrFromString("Choose the folder with RESOURCE.MAP, RESOURCE.001 and SCRANTIC.SCR:")
	name := make([]uint16, windows.MAX_PATH)
	info := struct {
		owner       uintptr
		root        uintptr
		displayName *uint16
		title       *uint16
		flags       uint32
		callback    uintptr
		param       uintptr
		image       int32
	}{owner: owner, displayName: &name[0], title: title, flags: 0x1 | 0x40 | 0x200} // only folders, new style, no "Make New Folder"
	list, _, _ := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&info)))
	if list == 0 {
		return "", false
	}
	defer procCoTaskMemFree.Call(list)
	path := make([]uint16, windows.MAX_PATH)
	if ok, _, _ := procSHGetPathFromIDListW.Call(list, uintptr(unsafe.Pointer(&path[0]))); ok == 0 {
		return "", false
	}
	return windows.UTF16ToString(path), true
}

// message shows text in a message box over owner.
func message(owner uintptr, text string) {
	title, _ := windows.UTF16PtrFromString("Johnny Castaway HD")
	p, _ := windows.UTF16PtrFromString(text)
	windows.MessageBox(windows.HWND(owner), p, title, windows.MB_OK|windows.MB_ICONWARNING)
}
