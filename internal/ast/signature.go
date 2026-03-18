package ast

import (
	"fmt"
	"sort"
	"strings"
)

func (d FuncDecl) EffectNames() []string {
	if len(d.Effects) == 0 {
		return nil
	}
	out := make([]string, 0, len(d.Effects))
	for _, effect := range d.Effects {
		out = append(out, effect.Name)
	}
	return out
}

func (d FuncDecl) CanonicalEffectNames() []string {
	names := d.EffectNames()
	sort.Strings(names)
	return names
}

func (d FuncDecl) Signature() string {
	var b strings.Builder
	b.WriteString("fn ")
	if d.Receiver != nil {
		fmt.Fprintf(&b, "(%s:%s) ", d.Receiver.Name, d.Receiver.Type)
	}
	fmt.Fprintf(&b, "%s(", d.Name)
	for i, param := range d.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%s:%s", param.Name, param.Type)
	}
	fmt.Fprintf(&b, ") -> %s", d.ReturnType)
	for _, effect := range d.CanonicalEffectNames() {
		fmt.Fprintf(&b, " !%s", effect)
	}
	return b.String()
}
