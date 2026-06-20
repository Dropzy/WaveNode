package artistmeta

import "strings"

var reusableLicenseNames = map[string]struct{}{
	"cc0":                          {},
	"cc-by":                        {},
	"cc-by-sa":                     {},
	"public domain":                {},
	"pd":                           {},
	"copyrighted free use":         {},
	"attribution":                  {},
	"creative commons attribution": {},
}

func IsReusableLicense(name string) bool {
	normalized := normalizeLicense(name)
	if normalized == "" {
		return false
	}
	if _, ok := reusableLicenseNames[normalized]; ok {
		return true
	}
	compact := strings.ReplaceAll(normalized, " ", "-")
	return strings.HasPrefix(normalized, "cc-by-") ||
		strings.HasPrefix(compact, "cc-by-") ||
		strings.HasPrefix(normalized, "cc by ") ||
		strings.Contains(normalized, "creative commons attribution") ||
		strings.Contains(normalized, "public domain")
}

func BuildAttribution(authorName, licenseName, sourcePageURL string) string {
	authorName = strings.TrimSpace(authorName)
	licenseName = strings.TrimSpace(licenseName)
	sourcePageURL = strings.TrimSpace(sourcePageURL)

	if isPublicDomainLicense(licenseName) {
		if sourcePageURL == "" {
			return "Public domain image"
		}
		return "Public domain image via " + sourcePageURL
	}

	parts := make([]string, 0, 3)
	if authorName != "" {
		parts = append(parts, authorName)
	} else {
		parts = append(parts, "Unknown author")
	}
	if licenseName != "" {
		parts = append(parts, licenseName)
	}
	if sourcePageURL != "" {
		parts = append(parts, sourcePageURL)
	}
	return strings.Join(parts, " · ")
}

func isPublicDomainLicense(name string) bool {
	normalized := normalizeLicense(name)
	return normalized == "cc0" || normalized == "pd" || strings.Contains(normalized, "public domain")
}

func normalizeLicense(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", "-")
	name = strings.Join(strings.Fields(name), " ")
	name = strings.TrimPrefix(name, "creative commons ")
	return name
}
