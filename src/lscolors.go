package main

// LS_COLORS support.
//
// The original script invented its own palette (group, dir, link, notable,
// warning, ...). That is still the default here, but if the user already has
// LS_COLORS (set by dircolors, by their shell, or by a theme), the few standard
// keys are honoured so that sl does not fight with the rest of the terminal:
//
//	di  directory            ln  symbolic link        ex  executable
//	or  orphan (broken link) mi  missing                fi  plain file
//	*.ext  per-extension colours
//
// Keys that sl has no equivalent for are ignored.

import (
	"os"
	"strings"
)

// lsExt maps a lowercase suffix (".c") to a style.
var lsExt = map[string]string{}

func parseLSColors(spec string) {
	for _, item := range strings.Split(spec, ":") {
		if item == "" {
			continue
		}
		eq := strings.IndexByte(item, '=')
		if eq <= 0 {
			continue
		}
		key, val := item[:eq], item[eq+1:]
		if val == "" || val == "0" || val == "00" {
			val = "" // "reset": no styling
		}
		switch key {
		case "di":
			styleMap["dir"] = normalizeSGR(val)
		case "ln":
			styleMap["link"] = normalizeSGR(val)
		case "ex":
			styleMap["exec"] = normalizeSGR(val)
		case "or":
			styleMap["broken"] = normalizeSGR(val)
		case "fi", "no", "rs", "mi", "pi", "so", "bd", "cd", "su", "sg", "tw", "ow", "st", "do":
			// no exact counterpart; deliberately ignored
		default:
			if strings.HasPrefix(key, "*.") && val != "" {
				lsExt[strings.ToLower(key[1:])] = normalizeSGR(val)
			}
		}
	}
}

// applyEnvColors reads LS_COLORS. NO_COLOR is handled in main.
func applyEnvColors() {
	if spec := os.Getenv("LS_COLORS"); spec != "" {
		parseLSColors(spec)
	}
}

// styleForExt returns the LS_COLORS style for a name, if the user set one.
func styleForExt(name string) (string, bool) {
	if len(lsExt) == 0 {
		return "", false
	}
	ext := strings.ToLower(tclExt(name))
	if s, ok := lsExt[ext]; ok {
		return s, true
	}
	return "", false
}
