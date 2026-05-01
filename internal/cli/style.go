package cli

import (
	"fmt"
	"io"
	"os"
)

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiDim    = "\x1b[2m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiBlue   = "\x1b[34m"
	ansiCyan   = "\x1b[36m"
)

type styler struct {
	color bool
}

func newStyler(w io.Writer) styler {
	if os.Getenv("NO_COLOR") != "" {
		return styler{}
	}
	if force := os.Getenv("FORCE_COLOR"); force != "" && force != "0" {
		return styler{color: true}
	}

	file, ok := w.(*os.File)
	if !ok {
		return styler{}
	}
	info, err := file.Stat()
	if err != nil {
		return styler{}
	}
	return styler{color: info.Mode()&os.ModeCharDevice != 0}
}

func (style styler) paint(code string, value string) string {
	if !style.color {
		return value
	}
	return code + value + ansiReset
}

func (style styler) bold(value string) string {
	return style.paint(ansiBold, value)
}

func (style styler) dim(value string) string {
	return style.paint(ansiDim, value)
}

func (style styler) red(value string) string {
	return style.paint(ansiRed, value)
}

func (style styler) green(value string) string {
	return style.paint(ansiGreen, value)
}

func (style styler) yellow(value string) string {
	return style.paint(ansiYellow, value)
}

func (style styler) blue(value string) string {
	return style.paint(ansiBlue, value)
}

func (style styler) cyan(value string) string {
	return style.paint(ansiCyan, value)
}

func (style styler) errorf(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "%s %s\n", style.red("error:"), fmt.Sprintf(format, args...))
}
