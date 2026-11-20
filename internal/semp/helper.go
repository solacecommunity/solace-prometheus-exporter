package semp

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
)

func mapItems(items []string, translateMap map[string]string) ([]string, error) {
	validRawItems := make(map[string]bool, len(translateMap))
	translated := make([]string, 0, len(items))
	validItems := make([]string, 0, len(translateMap)*2)

	for key, rawItem := range translateMap {
		validRawItems[rawItem] = true
		validItems = append(validItems, key, rawItem)
	}

	for _, item := range items {
		if translatedItem, ok := translateMap[item]; ok {
			translated = append(translated, translatedItem)
		} else if _, ok := validRawItems[item]; ok {
			translated = append(translated, item)
		} else {
			return nil, fmt.Errorf(
				"item \"%s\" is not valid. Pleaee choose from: %s",
				item,
				strings.Join(validItems, ","),
			)
		}
	}

	return translated, nil
}

func queryEscape(raw string) string {
	if len(strings.TrimSpace(raw)) == 0 {
		return raw
	}

	if strings.Contains(raw, "%") {
		// Seams already to be url encode
		return raw
	}

	return url.QueryEscape(raw)
}

// semp1Escape escapes a value, typically a VPN or item filter from the scrape request, for use as text content in a
// SEMP v1 command, so it cannot close its element and alter the command. Well-formed XML text decodes to the same
// value, so the broker receives exactly what it received before for every filter that was valid XML before.
func semp1Escape(value string) string {
	var sb strings.Builder
	_ = xml.EscapeText(&sb, []byte(value)) // writing to a strings.Builder cannot fail
	return sb.String()
}
