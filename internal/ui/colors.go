package ui

// Color codes for terminal output
const (
	ColorReset = "\033[0m"
	ColorBold  = "\033[1m"
	ColorDim   = "\033[2m"

	// Foreground colors
	ColorBlack   = "\033[30m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"

	// Bright foreground colors
	ColorBrightBlack   = "\033[90m"
	ColorBrightRed     = "\033[91m"
	ColorBrightGreen   = "\033[92m"
	ColorBrightYellow  = "\033[93m"
	ColorBrightBlue    = "\033[94m"
	ColorBrightMagenta = "\033[95m"
	ColorBrightCyan    = "\033[96m"
	ColorBrightWhite   = "\033[97m"
)

// Colorize wraps text with the given color code.
func Colorize(text, color string) string {
	return color + text + ColorReset
}

// Bold makes text bold.
func Bold(text string) string {
	return ColorBold + text + ColorReset
}

// Dim makes text dim/faded.
func Dim(text string) string {
	return ColorDim + text + ColorReset
}

// Semantic color functions for common use cases
func SuccessText(text string) string {
	return Colorize(text, ColorGreen)
}

func ErrorText(text string) string {
	return Colorize(text, ColorRed)
}

func WarningText(text string) string {
	return Colorize(text, ColorYellow)
}

func InfoText(text string) string {
	return Colorize(text, ColorCyan)
}

func HighlightText(text string) string {
	return Colorize(text, ColorBrightBlue)
}
