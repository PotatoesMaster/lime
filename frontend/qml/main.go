// Copyright 2013 The lime Authors.
// Use of this source code is governed by a 2-clause
// BSD-style license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"code.google.com/p/log4go"
	"fmt"
	"github.com/howeyc/fsnotify"
	"github.com/limetext/gopy/lib"
	"github.com/limetext/lime/backend"
	_ "github.com/limetext/lime/backend/commands"
	"github.com/limetext/lime/backend/render"
	"github.com/limetext/lime/backend/sublime"
	"github.com/limetext/lime/backend/textmate"
	"github.com/limetext/lime/backend/util"
	. "github.com/quarnster/util/text"
	"gopkg.in/qml.v0"
	"image/color"
	"io"
	"math"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	scheme *textmate.Theme
	blink  bool

	// http://qt-project.org/doc/qt-5.1/qtcore/qt.html#Key-enum
	lut = map[int]backend.Key{
		0x01000000: backend.Escape,
		0x01000001: '\t',
		// 0x01000002 // Qt::Key_Backtab
		0x01000003: backend.Backspace,
		0x01000004: backend.Enter,
		0x01000005: backend.KeypadEnter,
		0x01000006: backend.Insert,
		0x01000007: backend.Delete,
		0x01000008: backend.Break,
		// 0x01000009 // Qt::Key_Print
		// 0x0100000a // Qt::Key_SysReq
		// 0x0100000b // Qt::Key_Clear
		0x01000010: backend.Home,
		0x01000011: backend.End,
		0x01000012: backend.Left,
		0x01000013: backend.Up,
		0x01000014: backend.Right,
		0x01000015: backend.Down,
		0x01000016: backend.PageUp,
		0x01000017: backend.PageDown,
		// 0x01000020 // Qt::Key_Shift
		// 0x01000021 // Qt::Key_Control On Mac OS X, this corresponds to the Command keys.
		// 0x01000022 // Qt::Key_Meta On Mac OS X, this corresponds to the Control keys. On Windows keyboards, this key is mapped to the Windows key.
		// 0x01000023 // Qt::Key_Alt
		// 0x01001103 // Qt::Key_AltGr On Windows, when the KeyDown event for this key is sent, the Ctrl+Alt modifiers are also set.
		// 0x01000024 // Qt::Key_CapsLock
		// 0x01000025 // Qt::Key_NumLock
		// 0x01000026 // Qt::Key_ScrollLock
		0x01000030: backend.F1,
		0x01000031: backend.F2,
		0x01000032: backend.F3,
		0x01000033: backend.F4,
		0x01000034: backend.F5,
		0x01000035: backend.F6,
		0x01000036: backend.F7,
		0x01000037: backend.F8,
		0x01000038: backend.F9,
		0x01000039: backend.F10,
		0x0100003a: backend.F11,
		0x0100003b: backend.F12,
		// 0x01000053 // Qt::Key_Super_L
		// 0x01000054 // Qt::Key_Super_R
		// 0x01000055 // Qt::Key_Menu
		// 0x01000056 // Qt::Key_Hyper_L
		// 0x01000057 // Qt::Key_Hyper_R
		// 0x01000058 // Qt::Key_Help
		// 0x01000059 // Qt::Key_Clear
		// 0x01000060 // Qt::Key_Direction_R
		0x20: ' ',
		0x21: '!',
		0x22: '"',
		0x23: '#',
		0x24: '$',
		0x25: '%',
		0x26: '&',
		0x27: '\'',
		0x28: '(',
		0x29: ')',
		0x2a: '*',
		0x2b: '+',
		0x2c: ',',
		0x2d: '-',
		0x2e: '.',
		0x2f: '/',
		0x30: '0',
		0x31: '1',
		0x32: '2',
		0x33: '3',
		0x34: '4',
		0x35: '5',
		0x36: '6',
		0x37: '7',
		0x38: '8',
		0x39: '9',
		0x3a: ':',
		0x3b: ';',
		0x3c: '<',
		0x3d: '=',
		0x3e: '>',
		0x3f: '?',
		0x40: '@',
		0x41: 'a',
		0x42: 'b',
		0x43: 'c',
		0x44: 'd',
		0x45: 'e',
		0x46: 'f',
		0x47: 'g',
		0x48: 'h',
		0x49: 'i',
		0x4A: 'j',
		0x4B: 'k',
		0x4C: 'l',
		0x4d: 'm',
		0x4e: 'n',
		0x4f: 'o',
		0x50: 'p',
		0x51: 'q',
		0x52: 'r',
		0x53: 's',
		0x54: 't',
		0x55: 'u',
		0x56: 'v',
		0x57: 'w',
		0x58: 'x',
		0x59: 'y',
		0x5a: 'z',
		0x5b: '[',
		0x5c: '\\',
		0x5d: ']',
		0x5e: '°', // Qt::Key_AsciiCircum
		0x5f: '_', // Qt::Key_Underscore
		0x60: '`', // Qt::Key_QuoteLeft
		0x7b: '{', // Qt::Key_BraceLeft
		0x7c: '|', // Qt::Key_Bar
		0x7d: '}', // Qt::Key_BraceRight
		0x7e: '~', // Qt::Key_AsciiTilde
		// 0x0a0: '', // Qt::Key_nobreakspace
		// 0x0a1: '', // Qt::Key_exclamdown
		// 0x0a2: '', // Qt::Key_cent
		// 0x0a3: '', // Qt::Key_sterling
		// 0x0a4: '', // Qt::Key_currency
		// 0x0a5: '', // Qt::Key_yen
		// 0x0a6: '', // Qt::Key_brokenbar
		// 0x0a7: '', // Qt::Key_section
		// 0x0a8: '', // Qt::Key_diaeresis
		// 0x0a9: '', // Qt::Key_copyright
		// 0x0aa: '', // Qt::Key_ordfeminine
		// 0x0ab: '', // Qt::Key_guillemotleft
		// 0x0ac: '', // Qt::Key_notsign
		// 0x0ad: '', // Qt::Key_hyphen
		// 0x0ae: '', // Qt::Key_registered
		// 0x0af: '', // Qt::Key_macron
		0x0b0: '°', // Qt::Key_degree
		// 0x0b1: '', // Qt::Key_plusminus
		0x0b2: '²', // Qt::Key_twosuperior
		0x0b3: '³', // Qt::Key_threesuperior
		0x0b4: '´', // Qt::Key_acute
		// 0x0b5: '', // Qt::Key_mu
		// 0x0b6: '', // Qt::Key_paragraph
		// 0x0b7: '', // Qt::Key_periodcentered
		// 0x0b8: '', // Qt::Key_cedilla
		// 0x0b9: '', // Qt::Key_onesuperior
		// 0x0ba: '', // Qt::Key_masculine
		// 0x0bb: '', // Qt::Key_guillemotright
		// 0x0bc: '', // Qt::Key_onequarter
		// 0x0bd: '', // Qt::Key_onehalf
		// 0x0be: '', // Qt::Key_threequarters
		// 0x0bf: '', // Qt::Key_questiondown
		// 0x0c0: '', // Qt::Key_Agrave
		// 0x0c1: '', // Qt::Key_Aacute
		// 0x0c2: '', // Qt::Key_Acircumflex
		// 0x0c3: '', // Qt::Key_Atilde
		0x0c4: 'ä', // Qt::Key_Adiaeresis
		// 0x0c5: '', // Qt::Key_Aring
		// 0x0c6: '', // Qt::Key_AE
		// 0x0c7: '', // Qt::Key_Ccedilla
		// 0x0c8: '', // Qt::Key_Egrave
		// 0x0c9: '', // Qt::Key_Eacute
		// 0x0ca: '', // Qt::Key_Ecircumflex
		// 0x0cb: '', // Qt::Key_Ediaeresis
		// 0x0cc: '', // Qt::Key_Igrave
		// 0x0cd: '', // Qt::Key_Iacute
		// 0x0ce: '', // Qt::Key_Icircumflex
		// 0x0cf: '', // Qt::Key_Idiaeresis
		// 0x0d0: '', // Qt::Key_ETH
		// 0x0d1: '', // Qt::Key_Ntilde
		// 0x0d2: '', // Qt::Key_Ograve
		// 0x0d3: '', // Qt::Key_Oacute
		// 0x0d4: '', // Qt::Key_Ocircumflex
		// 0x0d5: '', // Qt::Key_Otilde
		0x0d6: 'ö', // Qt::Key_Odiaeresis
		// 0x0d7: '', // Qt::Key_multiply
		// 0x0d8: '', // Qt::Key_Ooblique
		// 0x0d9: '', // Qt::Key_Ugrave
		// 0x0da: '', // Qt::Key_Uacute
		// 0x0db: '', // Qt::Key_Ucircumflex
		0x0dc: 'ü', // Qt::Key_Udiaeresis
		// 0x0dd: '', // Qt::Key_Yacute
		// 0x0de: '', // Qt::Key_THORN
		// 0x0df: '', // Qt::Key_ssharp
		// 0x0f7: '', // Qt::Key_division
	}
)

const (
	console_height  = 20
	render_chan_len = 2

	// http://qt-project.org/doc/qt-5.1/qtcore/qt.html#KeyboardModifier-enum
	shift_mod  = 0x02000000
	ctrl_mod   = 0x04000000
	alt_mod    = 0x08000000
	meta_mod   = 0x10000000
	keypad_mod = 0x20000000
)

type (
	// keeping track of frontend state
	qmlfrontend struct {
		status_message string
		lock           sync.Mutex
		windows        map[*backend.Window]*frontendWindow
		Console        *frontendView
		qmlDispatch    chan qmlDispatch
	}
	// Used for batching qml.Changed calls
	qmlDispatch struct{ value, field interface{} }

	// This allows us to trigger a qml.Changed on a specific
	// line in the view so that only it is
	// re-rendered by qml
	lineStruct struct {
		Text string
	}
	// A helper glue structure connecting the backend Window with
	// the qml.Window
	frontendWindow struct {
		bw     *backend.Window
		Len    int
		views  []*frontendView
		window *qml.Window
	}

	// A helper glue structure connecting the backend View
	// with the qml code that then ends up rendering it.
	frontendView struct {
		bv            *backend.View
		Len           int
		FormattedLine []*lineStruct
		Title         lineStruct
	}
)

var (
	t *qmlfrontend
)

func htmlcol(c render.Colour) string {
	return fmt.Sprintf("%02X%02X%02X", c.R, c.G, c.B)
}

func (fw *frontendWindow) launch(wg *sync.WaitGroup, component qml.Object) {
	wg.Add(1)
	fw.window = component.CreateWindow(nil)
	fw.window.Show()
	fw.window.Set("myWindow", fw)

	go func() {
		fw.window.Wait()
		wg.Done()
	}()
}

func (fw *frontendWindow) View(idx int) *frontendView {
	return fw.views[idx]
}
func (t *qmlfrontend) Window(w *backend.Window) *frontendWindow {
	return t.windows[w]
}

func (t *qmlfrontend) Show(v *backend.View, r Region) {
	// TODO
}

func (t *qmlfrontend) VisibleRegion(v *backend.View) Region {
	// TODO
	return Region{0, v.Buffer().Size()}
}

func (t *qmlfrontend) StatusMessage(msg string) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.status_message = msg
}

func (t *qmlfrontend) ErrorMessage(msg string) {
	log4go.Error(msg)
}

// TODO(q): Actually show a dialog
func (t *qmlfrontend) MessageDialog(msg string) {
	log4go.Info(msg)
}

// TODO(q): Actually show a dialog
func (t *qmlfrontend) OkCancelDialog(msg, ok string) bool {
	log4go.Info(msg, ok)
	return false
}

func (t *qmlfrontend) scroll(b Buffer, pos, delta int) {
	t.Show(backend.GetEditor().Console(), Region{b.Size(), b.Size()})
}

func (fv *frontendView) Line(index int) *lineStruct {
	return fv.FormattedLine[index]
}

func (fv *frontendView) Setting(name string) interface{} {
	return fv.Back().Settings().Get(name, nil)
}

func (fv *frontendView) Back() *backend.View {
	return fv.bv
}
func (fw *frontendWindow) Back() *backend.Window {
	return fw.bw
}

// Apparently calling qml.Changed also triggers a re-draw, meaning that typed text is at the
// mercy of how quick Qt happens to be rendering.
// Try setting batching_enabled = false to see the effects of non-batching
func (t *qmlfrontend) qmlBatchLoop() {
	queue := make(map[qmlDispatch]bool)
	t.qmlDispatch = make(chan qmlDispatch, 1000)
	for {
		if len(queue) > 0 {
			select {
			case <-time.After(time.Millisecond * 20):
				// Nothing happened for 20 milliseconds, so dispatch all queued changes
				for k := range queue {
					qml.Changed(k.value, k.field)
				}
				queue = make(map[qmlDispatch]bool)
			case d := <-t.qmlDispatch:
				queue[d] = true
			}
		} else {
			queue[<-t.qmlDispatch] = true
		}
	}
}

const batching_enabled = true

func (t *qmlfrontend) qmlChanged(value, field interface{}) {
	if !batching_enabled {
		qml.Changed(value, field)
	} else {
		t.qmlDispatch <- qmlDispatch{value, field}
	}
}

func (fv *frontendView) bufferChanged(buf Buffer, pos, delta int) {
	prof := util.Prof.Enter("frontendView.bufferChanged")
	defer prof.Exit()
	row1, _ := buf.RowCol(pos)
	row2, _ := buf.RowCol(pos + delta)
	if row1 > row2 {
		row1, row2 = row2, row1
	}

	for i := row1; i <= row2; i++ {
		fv.formatLine(i)
	}
}

func (fv *frontendView) formatLine(line int) {
	prof := util.Prof.Enter("frontendView.formatLine")
	defer prof.Exit()
	buf := bytes.NewBuffer(nil)
	vr := fv.bv.Buffer().Line(fv.bv.Buffer().TextPoint(line, 0))
	for line >= len(fv.FormattedLine) {
		fv.FormattedLine = append(fv.FormattedLine, &lineStruct{Text: ""})
		fv.Len = len(fv.FormattedLine)
		t.qmlChanged(fv, &fv.Len)
	}
	if vr.Size() == 0 {
		// TODO: draw cursor if here
		if fv.FormattedLine[line].Text != "" {
			fv.FormattedLine[line].Text = ""
			t.qmlChanged(fv.FormattedLine[line], fv.FormattedLine[line])
		}
		return
	}
	recipie := fv.bv.Transform(scheme, vr).Transcribe()
	highlight_line := false
	if b, ok := fv.bv.Settings().Get("highlight_line", highlight_line).(bool); ok {
		highlight_line = b
	}
	lastEnd := vr.Begin()

	for _, reg := range recipie {
		if lastEnd != reg.Region.Begin() {
			fmt.Fprintf(buf, "<span>%s</span>", fv.bv.Buffer().Substr(Region{lastEnd, reg.Region.Begin()}))
		}
		fmt.Fprintf(buf, "<span style=\"white-space:pre; color:#%s; background:#%s\">%s</span>", htmlcol(reg.Flavour.Foreground), htmlcol(reg.Flavour.Background), fv.bv.Buffer().Substr(reg.Region))
		lastEnd = reg.Region.End()
	}
	if lastEnd != vr.End() {
		io.WriteString(buf, fv.bv.Buffer().Substr(Region{lastEnd, vr.End()}))
	}

	str := buf.String()

	if fv.FormattedLine[line].Text != str {
		fv.FormattedLine[line].Text = str
		t.qmlChanged(fv.FormattedLine[line], fv.FormattedLine[line])
	}
}

// Converting mouse click position to text position
// and adding appropriate region to view regionsets
// row is the line we are x is mouse x position and
// w is line width we will calculate x/w and multiply
// it to text length then rounding for finding closest
// text position to mouse click
func (fv *frontendView) LayoutToText(row, x, w int) {
	log4go.Info("%q", fv.Line(row))
	r := float64(x) / float64(w)
	p := fv.bv.Buffer().TextPoint(row, 0)
	lineR := fv.bv.Buffer().Line(p)
	line := fv.bv.Buffer().Substr(lineR)
	if x > w {
		p += len(line)
		fv.bv.Sel().Add(Region{p, p})
		return
	}

	tab := ""
	tab_size, ok := fv.bv.Settings().Get("tab_size", 4).(int)
	if !ok {
		tab_size = 4
	}
	for i := 0; i < tab_size; i++ {
		tab += " "
	}
	line = strings.Replace(line, "\t", tab, -1)

	col := int(math.Floor(float64(len(line))*r + 0.5))
	p = p + col
	fv.bv.Sel().Add(Region{p, p})
}

func (t *qmlfrontend) DefaultBg() color.RGBA {
	c := scheme.Spice(&render.ViewRegions{})
	c.Background.A = 0xff
	return color.RGBA(c.Background)
}

func (t *qmlfrontend) DefaultFg() color.RGBA {
	c := scheme.Spice(&render.ViewRegions{})
	c.Foreground.A = 0xff
	return color.RGBA(c.Foreground)
}

func (t *qmlfrontend) loop() (err error) {

	backend.OnNew.Add(func(v *backend.View) {
		fv := &frontendView{bv: v}
		v.Buffer().AddCallback(fv.bufferChanged)
		v.Settings().AddOnChange("blah", func(name string) {
			if name == "lime.syntax.updated" {
				// force redraw, as the syntax regions might have changed...
				for i := range fv.FormattedLine {
					fv.formatLine(i)
				}
			}
		})

		fv.Title.Text = v.Buffer().FileName()
		if len(fv.Title.Text) == 0 {
			fv.Title.Text = "untitled"
		}

		w2 := t.windows[v.Window()]
		w2.views = append(w2.views, fv)
		w2.Len = len(w2.views)
		t.qmlChanged(w2, &w2.Len)
	})

	backend.OnLoad.Add(func(v *backend.View) {
		w2 := t.windows[v.Window()]
		i := 0
		for i, _ = range w2.views {
			if w2.views[i].bv == v {
				break
			}
		}
		v2 := w2.views[i]
		v2.Title.Text = v.Buffer().FileName()
		t.qmlChanged(v2, &v2.Title)
	})

	ed := backend.GetEditor()
	ed.SetFrontend(t)
	ed.LogInput(false)
	ed.LogCommands(false)
	c := ed.Console()
	t.Console = &frontendView{bv: c}
	c.Buffer().AddCallback(t.Console.bufferChanged)
	c.Buffer().AddCallback(t.scroll)

	const qmlMainFile = "main.qml"
	var (
		engine    *qml.Engine
		component qml.Object
		// WaitGroup keeping track of open windows
		wg sync.WaitGroup
	)

	// create and setup a new engine, destroying
	// the old one if one exists.
	//
	// This is needed to re-load qml files to get
	// the new file contents from disc as otherwise
	// the old file would still be what is referenced.
	newEngine := func() (err error) {
		if engine != nil {
			log4go.Debug("calling destroy")
			// TODO(.): calling this appears to make the editor *very* crash-prone, just let it leak for now
			// engine.Destroy()
			engine = nil
		}
		log4go.Debug("calling newEngine")
		engine = qml.NewEngine()
		log4go.Debug("setvar frontend")
		engine.Context().SetVar("frontend", t)
		log4go.Debug("setvar editor")
		engine.Context().SetVar("editor", backend.GetEditor())

		log4go.Debug("loadfile")
		component, err = engine.LoadFile(qmlMainFile)
		return
	}
	if err := newEngine(); err != nil {
		log4go.Error(err)
	}

	backend.OnNewWindow.Add(func(w *backend.Window) {
		fw := &frontendWindow{bw: w}
		t.windows[w] = fw
		if component != nil {
			fw.launch(&wg, component)
		}
	})

	// TODO: should be done backend side
	if sc, err := textmate.LoadTheme("../../3rdparty/bundles/TextMate-Themes/GlitterBomb.tmTheme"); err != nil {
		log4go.Error(err)
	} else {
		scheme = sc
	}

	defer func() {
		fmt.Println(util.Prof)
	}()

	w := ed.NewWindow()
	v := w.OpenFile("main.go", 0)
	// TODO: should be done backend side
	v.Settings().Set("syntax", "../../3rdparty/bundles/go.tmbundle/Syntaxes/Go.tmLanguage")
	v.Sel().Clear()
	v.Sel().Add(Region{0, 0})
	v = w.OpenFile("../../backend/editor.go", 0)
	// TODO: should be done backend side
	v.Settings().Set("syntax", "../../3rdparty/bundles/go.tmbundle/Syntaxes/Go.tmLanguage")
	v.Sel().Clear()
	v.Sel().Add(Region{0, 0})

	ed.Init()
	sublime.Init()

	watch, err := fsnotify.NewWatcher()
	if err != nil {
		log4go.Error("Unable to create file watcher: %s", err)
		return
	}
	defer watch.Close()
	watch.Watch(".")
	defer watch.RemoveWatch(".")

	reloadRequested := false

	go func() {
		for {
			select {
			case ev := <-watch.Event:
				if ev != nil && strings.HasSuffix(ev.Name, ".qml") && ev.IsModify() && !ev.IsAttrib() {
					reloadRequested = true
					// Close all open windows to de-reference all
					// qml objects
					for _, v := range t.windows {
						if v.window != nil {
							v.window.Hide()
							v.window.Destroy()
							v.window = nil
						}
					}
				}
			}
		}
	}()

	for {
		// Reset reload status
		reloadRequested = false

		log4go.Debug("Waiting for all windows to close")
		// wg would be the WaitGroup all windows belong to, so first we wait for
		// all windows to close.
		wg.Wait()
		log4go.Debug("All windows closed. reloadRequest: %v", reloadRequested)
		// then we check if there's a reload request in the pipe
		if !reloadRequested || len(t.windows) == 0 {
			// This would be a genuine exit; all windows closed by the user
			break
		}

		// *We* closed all windows because we want to reload freshly changed qml
		// files.
		for {
			log4go.Debug("Calling newEngine")
			if err := newEngine(); err != nil {
				// Reset reload status
				reloadRequested = false
				log4go.Error(err)
				for !reloadRequested {
					// This loop allows us to re-try reloading
					// if there was an error in the file this time,
					// we just loop around again when we receive the next
					// reload request (ie on the next save of the file).
					time.Sleep(time.Second)
				}
				continue
			}
			log4go.Debug("break")
			break
		}
		log4go.Debug("re-launching all windows")
		// Succeeded loading the file, re-launch all windows
		for _, v := range t.windows {
			v.launch(&wg, component)
		}
	}
	return
}

func (t *qmlfrontend) HandleInput(keycode int, modifiers int) bool {
	log4go.Debug("qmlfrontend.HandleInput: key=%x, modifiers=%x", keycode, modifiers)
	shift := false
	alt := false
	ctrl := false
	super := false

	if key, ok := lut[keycode]; ok {
		ed := backend.GetEditor()

		if (modifiers & shift_mod) != 0 {
			shift = true
		}
		if (modifiers & alt_mod) != 0 {
			alt = true
		}
		if (modifiers & ctrl_mod) != 0 {
			if runtime.GOOS == "darwin" {
				super = true
			} else {
				ctrl = true
			}
		}
		if (modifiers & meta_mod) != 0 {
			if runtime.GOOS == "darwin" {
				ctrl = true
			} else {
				super = true
			}
		}

		ed.HandleInput(backend.KeyPress{Key: key, Shift: shift, Alt: alt, Ctrl: ctrl, Super: super})
		return true
	}
	return false
}

func main() {
	log4go.AddFilter("file", log4go.FINEST, log4go.NewConsoleLogWriter())
	defer func() {
		py.NewLock()
		py.Finalize()
	}()

	t = &qmlfrontend{windows: make(map[*backend.Window]*frontendWindow)}
	go t.qmlBatchLoop()
	qml.Init(nil)
	t.loop()
	//TODO: for qml.v1
	//	qml.Run(nil, t.loop)
}
