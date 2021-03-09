package main

import "github.com/rivo/tview"

type LogWindow struct {
	root     *app
	textView *tview.TextView
}

func newLogWindow(root *app, maxLines int) *LogWindow {
	textView := tview.NewTextView()
	textView.SetTitle("log")
	textView.SetBorder(true)
	textView.SetMaxLines(maxLines)
	textView.SetWordWrap(true)
	textView.SetWrap(true)
	return &LogWindow{
		root:     root,
		textView: textView,
	}
}

func (l *LogWindow) focus() {
	l.root.root.SetFocus(l.textView)
}

func (l *LogWindow) Write(buf []byte) (int, error) {
	return l.textView.Write(buf)
}
