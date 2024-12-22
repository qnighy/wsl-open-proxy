package xdgini

import (
	"slices"
	"strings"
	"unicode"
)

type Config struct {
	Groups map[string]*ConfigGroup
	EndRaw *RawLineStyle
}

type ConfigGroup struct {
	Entries map[string]*ConfigEntry
	Raws    []*RawLineStyle
}

type ConfigEntry struct {
	Value string
	Raws  []*RawLineStyle
}

type RawLineStyle struct {
	Order            int
	Line             string
	LeadingComments  []string
	TrailingComments []string
}

func ParseConfig(data string) *Config {
	config := &Config{
		Groups: map[string]*ConfigGroup{},
	}
	orderStep := 10
	currentOrder := orderStep
	var currentGroup *ConfigGroup
	var lastLine *RawLineStyle
	var pendingComments []string

	pos := 0
	for pos < len(data) {
		lineEnd := strings.IndexByte(data[pos:], '\n')
		if lineEnd < 0 {
			lineEnd = len(data)
		} else {
			lineEnd += pos + 1
		}
		line := data[pos:lineEnd]
		pos = lineEnd

		tLine := strings.TrimRightFunc(line, unicode.IsSpace)
		if tLine == "" && lastLine != nil {
			lastLine.TrailingComments = append(lastLine.TrailingComments, line)
			continue
		} else if strings.HasPrefix(tLine, "#") || tLine == "" {
			lastLine = nil
			pendingComments = append(pendingComments, line)
			continue
		}

		if strings.HasPrefix(tLine, "[") {
			groupName := parseGroupLine(line)
			if group, ok := config.Groups[groupName]; ok {
				// Duplicate group (not allowed spec-wise)
				currentGroup = group
			} else {
				currentGroup = &ConfigGroup{
					Entries: map[string]*ConfigEntry{},
				}
				config.Groups[groupName] = currentGroup
			}
			lastLine = &RawLineStyle{
				Order:            currentOrder,
				Line:             line,
				LeadingComments:  pendingComments,
				TrailingComments: nil,
			}
			currentGroup.Raws = append(currentGroup.Raws, lastLine)
			currentOrder += orderStep
			pendingComments = nil
		} else {
			key, value := parseKeyValueLine(line)
			if currentGroup == nil {
				// Introduce a dummy group
				currentGroup = &ConfigGroup{
					Entries: map[string]*ConfigEntry{},
					Raws: []*RawLineStyle{
						{
							Order: currentOrder,
							Line:  "",
						},
					},
				}
				currentOrder += orderStep
				config.Groups[""] = currentGroup
			}
			lastLine = &RawLineStyle{
				Order:            currentOrder,
				Line:             line,
				LeadingComments:  pendingComments,
				TrailingComments: nil,
			}
			pendingComments = nil
			if entry, ok := currentGroup.Entries[key]; ok {
				// Duplicate entry (not allowed spec-wise)
				// Prefer first occurrence
				entry.Raws = append(entry.Raws, lastLine)
			} else {
				entry := &ConfigEntry{
					Value: value,
					Raws:  []*RawLineStyle{lastLine},
				}
				currentGroup.Entries[key] = entry
			}
		}
	}
	config.EndRaw = &RawLineStyle{
		Order:            currentOrder,
		TrailingComments: pendingComments,
	}
	currentOrder += orderStep
	pendingComments = nil
	return config
}

func (c *Config) String() string {
	type ConfigSortItem struct {
		Order int
		Name  string
		Lines []string
	}
	var configItems []*ConfigSortItem
	for groupName, group := range c.Groups {
		type GroupSortEntry struct {
			Order int
			Group *ConfigGroup
			Entry *ConfigEntry
			Line  string
			Raw   *RawLineStyle
		}
		var groupItems []*GroupSortEntry
		for _, raw := range group.Raws {
			entry := &GroupSortEntry{
				Order: raw.Order,
				Group: group,
				Line:  raw.Line,
				Raw:   raw,
			}
			if parseGroupLine(entry.Line) != groupName {
				entry.Line = "[" + groupName + "]\n"
			}
			groupItems = append(groupItems, entry)
		}
		if len(group.Raws) == 0 {
			groupItems = append(groupItems, &GroupSortEntry{
				Order: c.EndRaw.Order,
				Group: group,
				Line:  "[" + groupName + "]\n",
			})
		}
		for entryKey, entry := range group.Entries {
			if len(entry.Raws) == 0 {
				groupItems = append(groupItems, &GroupSortEntry{
					Order: c.EndRaw.Order,
					Entry: entry,
					Line:  entryKey + "=" + entry.Value + "\n",
				})
				continue
			}

			_, origValue := parseKeyValueLine(entry.Raws[0].Line)
			if origValue != entry.Value {
				groupItems = append(groupItems, &GroupSortEntry{
					Order: entry.Raws[0].Order,
					Entry: entry,
					Line:  entryKey + "=" + entry.Value + "\n",
					Raw:   entry.Raws[0],
				})
				continue
			}

			for _, raw := range entry.Raws {
				groupItems = append(groupItems, &GroupSortEntry{
					Order: raw.Order,
					Entry: entry,
					Line:  raw.Line,
					Raw:   raw,
				})
			}
		}
		slices.SortStableFunc(groupItems, func(a, b *GroupSortEntry) int {
			if a.Order < b.Order {
				return -1
			} else if a.Order > b.Order {
				return 1
			}
			if a.Group != nil && b.Entry != nil {
				return -1
			} else if a.Entry != nil && b.Group != nil {
				return 1
			}
			if a.Entry != nil && b.Entry != nil {
				keyCmp := strings.Compare(a.Entry.Value, b.Entry.Value)
				if keyCmp != 0 {
					return keyCmp
				}
			}
			return 0
		})

		// Swap out the first header so that every entry is preceded by a header
		firstHeaderIdx := slices.IndexFunc(groupItems, func(e *GroupSortEntry) bool {
			return e.Group != nil
		})
		if firstHeaderIdx > 0 {
			firstHeader := groupItems[firstHeaderIdx]
			copy(groupItems[1:firstHeaderIdx+1], groupItems[:firstHeaderIdx])
			groupItems[0] = firstHeader
		}
		currentItem := &ConfigSortItem{
			Order: groupItems[0].Order,
			Name:  groupName,
		}
		if groupItems[0].Raw != nil {
			currentItem.Lines = append(currentItem.Lines, groupItems[0].Raw.LeadingComments...)
		}
		currentItem.Lines = append(currentItem.Lines, groupItems[0].Line)
		if groupItems[0].Raw != nil {
			currentItem.Lines = append(currentItem.Lines, groupItems[0].Raw.TrailingComments...)
		}
		for _, entry := range groupItems[1:] {
			if entry.Group != nil {
				configItems = append(configItems, currentItem)
				currentItem = &ConfigSortItem{
					Order: entry.Order,
					Name:  groupName,
				}
				if entry.Raw != nil {
					currentItem.Lines = append(currentItem.Lines, entry.Raw.LeadingComments...)
				}
				currentItem.Lines = append(currentItem.Lines, entry.Line)
				if entry.Raw != nil {
					currentItem.Lines = append(currentItem.Lines, entry.Raw.TrailingComments...)
				}
				continue
			}
			if entry.Raw != nil {
				currentItem.Lines = append(currentItem.Lines, entry.Raw.LeadingComments...)
			}
			currentItem.Lines = append(currentItem.Lines, entry.Line)
			if entry.Raw != nil {
				currentItem.Lines = append(currentItem.Lines, entry.Raw.TrailingComments...)
			}
		}
		configItems = append(configItems, currentItem)
	}

	slices.SortStableFunc(configItems, func(a, b *ConfigSortItem) int {
		if a.Order < b.Order {
			return -1
		} else if a.Order > b.Order {
			return 1
		}
		return strings.Compare(a.Name, b.Name)
	})

	for i, configItem := range configItems {
		if i > 0 && len(configItem.Lines) > 0 && configItem.Lines[0] == "" {
			// Instantiate dummy group header
			configItem.Lines[0] = "[]\n"
		}
	}

	var lines []string
	for _, item := range configItems {
		lines = append(lines, item.Lines...)
	}
	if c.EndRaw != nil {
		lines = append(lines, c.EndRaw.TrailingComments...)
	}
	for i, line := range lines {
		if i < len(lines)-1 && !strings.HasSuffix(line, "\n") && line != "" {
			lines[i] = line + "\n"
		}
	}
	return strings.Join(lines, "")
}

func parseGroupLine(line string) string {
	tLine := strings.TrimRightFunc(line, unicode.IsSpace)
	if strings.HasPrefix(tLine, "[") {
		return strings.TrimSuffix(strings.TrimPrefix(tLine, "["), "]")
	} else {
		return ""
	}
}

func parseKeyValueLine(line string) (key string, value string) {
	tLine := strings.TrimRightFunc(line, unicode.IsSpace)
	eqPos := strings.IndexByte(tLine, '=')
	if eqPos < 0 {
		return tLine, ""
	} else {
		return strings.TrimSpace(tLine[:eqPos]), strings.TrimSpace(tLine[eqPos+1:])
	}
}
