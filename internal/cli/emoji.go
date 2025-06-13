package cli

import (
	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/LogSum/internal/emoji"
)

// GetEmoji is a wrapper for the shared emoji package
func GetEmoji(key string) string {
	return emoji.GetEmoji(key)
}

// GetPatternEmoji returns emoji for pattern types with fallback support
func GetPatternEmoji(patternType common.PatternType) string {
	switch patternType {
	case common.PatternTypeError:
		return GetEmoji("error_pattern")
	case common.PatternTypeAnomaly:
		return GetEmoji("anomaly_pattern")
	case common.PatternTypePerformance:
		return GetEmoji("perf_pattern")
	case common.PatternTypeSecurity:
		return GetEmoji("security_pattern")
	default:
		return GetEmoji("pattern")
	}
}

// GetSeverityEmoji returns emoji for severity levels with fallback support
func GetSeverityEmoji(severity common.LogLevel) string {
	switch severity {
	case common.LevelFatal, common.LevelError:
		return GetEmoji("error")
	case common.LevelWarn:
		return GetEmoji("warning")
	case common.LevelInfo:
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
