package main

// Appearance setting values saved in hosas_config.json ("theme") and their
// labels in the Theme dropdown, in the same order.
const (
	themeLight  = "Light"
	themeDark   = "Dark"
	themeSystem = "System"
)

var themeValues = []string{themeLight, themeDark, themeSystem}
var themeLabels = []string{"Light Mode", "Dark Mode", "System Specified"}

// normalizeTheme maps saved or missing values to a known theme. Settings
// written before the Theme option existed keep the original light look.
func normalizeTheme(s string) string {
	for _, v := range themeValues {
		if s == v {
			return v
		}
	}
	return themeLight
}

func themeIndex(s string) int {
	s = normalizeTheme(s)
	for i, v := range themeValues {
		if s == v {
			return i
		}
	}
	return 0
}

// useDarkTheme resolves System Specified with the Windows app preference.
func useDarkTheme(theme string, systemPrefersDark bool) bool {
	switch normalizeTheme(theme) {
	case themeDark:
		return true
	case themeSystem:
		return systemPrefersDark
	}
	return false
}
