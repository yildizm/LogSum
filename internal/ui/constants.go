package ui

// ASCII art and symbols
const (
	LogoText = `
 ▄▄▄▄▄▄▄▄▄▄▄  ▄▄▄▄▄▄▄▄▄▄▄  ▄▄▄▄▄▄▄▄▄▄▄  ▄▄▄▄▄▄▄▄▄▄▄  ▄         ▄  ▄▄       ▄▄ 
▐░░░░░░░░░░░▌▐░░░░░░░░░░░▌▐░░░░░░░░░░░▌▐░░░░░░░░░░░▌▐░▌       ▐░▌▐░░▌     ▐░░▌
▐░█▀▀▀▀▀▀▀█░▌▐░█▀▀▀▀▀▀▀█░▌▐░█▀▀▀▀▀▀▀▀▀ ▐░█▀▀▀▀▀▀▀▀▀ ▐░▌       ▐░▌▐░▌░▌   ▐░▐░▌
▐░▌       ▐░▌▐░▌       ▐░▌▐░▌          ▐░▌          ▐░▌       ▐░▌▐░▌▐░▌ ▐░▌▐░▌
▐░▌       ▐░▌▐░▌       ▐░▌▐░▌ ▄▄▄▄▄▄▄▄ ▐░▌ ▄▄▄▄▄▄▄▄ ▐░▌       ▐░▌▐░▌ ▐░▐░▌ ▐░▌
▐░▌       ▐░▌▐░▌       ▐░▌▐░▌▐░░░░░░░░▌▐░▌▐░░░░░░░░▌▐░▌       ▐░▌▐░▌  ▐░▌  ▐░▌
▐░▌       ▐░▌▐░▌       ▐░▌▐░█▄▄▄▄▄▄▄▄▄ ▐░█▄▄▄▄▄▄▄▄▄ ▐░▌       ▐░▌▐░▌   ▀   ▐░▌
▐░▌       ▐░▌▐░▌       ▐░▌▐░░░░░░░░░░░▌▐░░░░░░░░░░░▌▐░▌       ▐░▌▐░▌       ▐░▌
▐░█▄▄▄▄▄▄▄█░▌▐░█▄▄▄▄▄▄▄█░▌ ▀▀▀▀▀▀▀▀▀█░▌ ▀▀▀▀▀▀▀▀▀█░▌▐░█▄▄▄▄▄▄▄█░▌▐░▌       ▐░▌
▐░░░░░░░░░░░▌▐░░░░░░░░░░░▌          ▐░▌          ▐░▌▐░░░░░░░░░░░▌▐░▌       ▐░▌
 ▀▀▀▀▀▀▀▀▀▀▀  ▀▀▀▀▀▀▀▀▀▀▀            ▀            ▀  ▀▀▀▀▀▀▀▀▀▀▀  ▀         ▀ 
`

	SimpleLogoText = `
╦  ╔═╗╔═╗╔═╗╦ ╦╔╦╗
║  ║ ║║ ╦╚═╗║ ║║║║
╩═╝╚═╝╚═╝╚═╝╚═╝╩ ╩
`

	SpinnerFrames = "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
)

// Box drawing characters
const (
	BoxTopLeft     = "┌"
	BoxTopRight    = "┐"
	BoxBottomLeft  = "└"
	BoxBottomRight = "┘"
	BoxHorizontal  = "─"
	BoxVertical    = "│"
	BoxTee         = "┬"
	BoxCross       = "┼"
)

// Status symbols
const (
	SymbolSuccess = "✓"
	SymbolWarning = "⚠"
	SymbolError   = "✗"
	SymbolInfo    = "ℹ"
	SymbolInsight = "💡"
	SymbolPattern = "🔍"
	SymbolTime    = "🕒"
)

// Fallback symbols for when emojis are disabled
const (
	FallbackSuccess = "[✓]"
	FallbackWarning = "[!]"
	FallbackError   = "[X]"
	FallbackInfo    = "[i]"
	FallbackInsight = "[*]"
	FallbackPattern = "[?]"
	FallbackTime    = "[T]"
)
