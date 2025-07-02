package termui

import "fmt"

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Bold   = "\033[1m"

	ClearLine = "\033[1A\033[K"
)

type TermUI struct {
	statusLine string
}

func (ui *TermUI) Printf(format string, a ...any) {
	fmt.Printf(ClearLine)
	fmt.Printf(format, a...)
	fmt.Println(ui.statusLine)
}

func (ui *TermUI) UpdateStatus(format string, a ...any) {
	if len(ui.statusLine) > 0 {
		fmt.Printf(ClearLine)
	}
	ui.statusLine = fmt.Sprintf(format, a...)
	if len(ui.statusLine) > 0 {
		fmt.Println(ui.statusLine)
	}
}

func (ui *TermUI) ClearStatus() {
	ui.UpdateStatus("")
}
