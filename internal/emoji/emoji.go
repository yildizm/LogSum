package emoji

// EmojiMap holds emoji and fallback mappings
var emojiMap = map[string][2]string{
	// [emoji, fallback]
	"error":            {"❌", "[ERR]"},
	"warning":          {"⚠️", "[WRN]"},
	"info":             {"ℹ️", "[INF]"},
	"success":          {"✅", "[OK]"},
	"insight":          {"💡", "[INS]"},
	"pattern":          {"🔍", "[PAT]"},
	"statistics":       {"📊", "[STATS]"},
	"recommendations":  {"📋", "[REC]"},
	"rocket":           {"🚀", "[LOG]"},
	"error_pattern":    {"🔴", "[ERR]"},
	"anomaly_pattern":  {"🟡", "[ANO]"},
	"perf_pattern":     {"⚠️", "[PRF]"},
	"security_pattern": {"🔒", "[SEC]"},
	"help":             {"❓", "[?]"},
	"target":           {"🎯", "[>]"},
	"brain":            {"🧠", "[AI]"},
	"tag":              {"🏷️", "[TAG]"},
	"scale":            {"⚖️", "[BAL]"},
	"door":             {"🚪", "[EXIT]"},
	"number":           {"🔢", "[#]"},
}

var emojiDisabled bool

// SetEmojiDisabled sets the global emoji disabled state
func SetEmojiDisabled(disabled bool) {
	emojiDisabled = disabled
	// Debug: uncomment to verify the setting is being applied
	// fmt.Printf("DEBUG: Emoji disabled set to: %v\n", disabled)
}

// IsEmojiDisabled returns the current emoji disabled state
func IsEmojiDisabled() bool {
	return emojiDisabled
}

// GetEmoji returns emoji or fallback based on no-emoji setting
func GetEmoji(key string) string {
	if mapping, exists := emojiMap[key]; exists {
		if emojiDisabled {
			return mapping[1] // fallback
		}
		return mapping[0] // emoji
	}
	return "[?]" // unknown key
}
