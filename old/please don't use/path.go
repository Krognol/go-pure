package pure

import "path/filepath"

type Path struct {
	Value string
}

func NewPath(val string) *Path {
	return &Path{val}
}

func (p *Path) Directory() string {
	return filepath.Dir(p.Value)
}

func (p *Path) Base() string {
	return filepath.Base(p.Value)
}

func (p *Path) FileExtension() string {
	return filepath.Ext(p.Value)
}

func (p *Path) VolumeName() string {
	return filepath.VolumeName(p.Value)
}
