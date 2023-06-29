package treew

import "strings"

type Style struct {
	Branch string
	Gap    string
	First  string
	Next   string
	Last   string
}

var DefaultStyle = BoxDrawStyle()

func ASCIIStyle() *Style {
	return &Style{
		Branch: "|   ",
		Gap:    "    ",
		First:  ",-- ",
		Next:   "+-- ",
		Last:   "`-- ",
	}
}

func BoxDrawStyle() *Style {
	return &Style{
		Branch: "│  ",
		Gap:    "   ",
		First:  "┌─ ",
		Next:   "├─ ",
		Last:   "└─ ",
	}
}

func ItemStyle() *Style {
	return &Style{
		Branch: "  ",
		Gap:    "  ",
		First:  "- ",
		Next:   "- ",
		Last:   "- ",
	}
}

type mode int

const (
	branching mode = iota + 1
	dangling
)

type Prefix struct {
	Style  *Style
	prefix []mode
	mode   mode
}

func (p *Prefix) Descend() *Prefix {
	switch p.mode {
	case 0:
		p.mode = branching
	default:
		p.prefix = append(p.prefix, p.mode)
	}
	p.mode = branching
	return p
}

func (p *Prefix) Ascend(up int) *Prefix {
	for up > 0 { // TODO eliminate loop
		if l := len(p.prefix); l > 0 {
			p.mode = p.prefix[l-1]
			p.prefix = p.prefix[:l-1]
		} else {
			p.mode = 0
			break
		}
		up--
	}
	return p
}

func (p *Prefix) First(s *Style) string {
	if p.mode == 0 {
		return ""
	}
	s = p.style(s)
	var sb strings.Builder
	p.branches(&sb, s)
	sb.WriteString(s.First)
	p.mode = branching
	return sb.String()
}

func (p *Prefix) Next(s *Style) string {
	if p.mode == 0 {
		return ""
	}
	s = p.style(s)
	var sb strings.Builder
	p.branches(&sb, s)
	sb.WriteString(s.Next)
	p.mode = branching
	return sb.String()
}

func (p *Prefix) Last(s *Style) string {
	if p.mode == 0 {
		return ""
	}
	s = p.style(s)
	var sb strings.Builder
	p.branches(&sb, s)
	sb.WriteString(s.Last)
	p.mode = dangling
	return sb.String()
}

func (p *Prefix) Cont(s *Style) string {
	if p.mode == 0 {
		return ""
	}
	s = p.style(s)
	var sb strings.Builder
	p.branches(&sb, s)
	switch p.mode {
	case branching:
		sb.WriteString(s.Branch)
	default:
		sb.WriteString(s.Gap)
	}
	return sb.String()
}

func (p *Prefix) style(s *Style) *Style {
	if s != nil {
		return s
	}
	if p.Style != nil {
		return p.Style
	}
	return DefaultStyle
}

func (p *Prefix) branches(sb *strings.Builder, s *Style) {
	for _, gap := range p.prefix {
		switch gap {
		case branching:
			sb.WriteString(s.Branch)
		default:
			sb.WriteString(s.Gap)
		}
	}
}
