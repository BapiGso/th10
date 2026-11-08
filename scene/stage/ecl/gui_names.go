package ecl

import (
	"strings"

	"th10/gamedata"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// spellDisplayNames pairs declarations by subroutine and occurrence, as
// docs/research/th10c_dat.md requires. The localized .ect is never executed:
// stage06.ect differs in instructions, and must not change game logic.
func spellDisplayNames(originalPath string, original *Program) map[string]string {
	data, err := gamedata.ReadFile(strings.TrimSuffix(originalPath, ".ecl") + ".ect")
	if err != nil {
		return nil
	}
	localized, err := LoadProgram(data)
	if err != nil {
		return nil
	}
	out := map[string]string{}
	jp, zh := japanese.ShiftJIS.NewDecoder(), simplifiedchinese.GBK.NewDecoder()
	for name, sub := range original.SubByName {
		other := localized.SubByName[name]
		if other == nil {
			continue
		}
		a, b := declarationNames(sub), declarationNames(other)
		if len(a) != len(b) {
			continue
		}
		for i := range a {
			key, err1 := jp.String(a[i])
			text, err2 := zh.String(b[i])
			if err1 == nil && err2 == nil && text != "" {
				out[key] = text
			}
		}
	}
	return out
}

func declarationNames(s *Sub) (out []string) {
	for _, in := range s.Instrs {
		switch in.Opcode {
		case opSpellBg, opSpellAlt, opSpell, opSpell2, opSpell3:
			if len(in.Params) >= 4 {
				out = append(out, in.Params[3].Str)
			}
		}
	}
	return out
}
