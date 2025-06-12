package cli

import (
	"github.com/yildizm/LogSum/internal/emoji"
	"github.com/yildizm/LogSum/internal/parser"
)

// GetEmoji is a wrapper for the shared emoji package
func GetEmoji(key string) string {
	return emoji.GetEmoji(key)
}

// GetPatternEmoji returns emoji for pattern types with fallback support
func GetPatternEmoji(patternType parser.PatternType) string {
	switch patternType {
	case parser.PatternTypeError:
		return GetEmoji("error_pattern")
	case parser.PatternTypeAnomaly:
		return GetEmoji("anomaly_pattern")
	case parser.PatternTypePerformance:
		return GetEmoji("perf_pattern")
	case parser.PatternTypeSecurity:
		return GetEmoji("security_pattern")
	default:
		return GetEmoji("pattern")
	}
}

// GetSeverityEmoji returns emoji for severity levels with fallback support
func GetSeverityEmoji(severity parser.LogLevel) string {
	switch severity {
	case parser.LevelFatal, parser.LevelError:
		return GetEmoji("error")
	case parser.LevelWarn:
		return GetEmoji("warning")
	case parser.LevelInfo:
		return GetEmoji("info")
	default:
		return GetEmoji("insight")
	}
}

// GetSymbol returns UI symbols with fallback support
func GetSymbol(symbolType string) string {
	return GetEmoji(symbolType)
}

// CreateConfidenceBar creates ASCII confidence bar with emoji fallback
func CreateConfidenceBar(confidence float64) string {
	barLength := int(confidence * 10) // 10 character bar

	if isEmojiDisabled() {
		filled := make([]rune, barLength)
		empty := make([]rune, 10-barLength)

		for i := range filled {
			filled[i] = '#'
		}
		for i := range empty {
			empty[i] = '-'
		}

		return "[" + string(filled) + string(empty) + "]"
	}

	filled := make([]rune, barLength)
	empty := make([]rune, 10-barLength)

	for i := range filled {
		filled[i] = '█'
	}
	for i := range empty {
		empty[i] = '░'
	}

	return string(filled) + string(empty)
}
