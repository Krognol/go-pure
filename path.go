package pure

import "path/filepath"

type Path struct {
	value string
}

func NewPath(val string) *Path {
	return &Path{val}
}

func (p *Path) Directory() string {
	return filepath.Dir(p.value)
}

func (p *Path) Base() string {
	return filepath.Base(p.value)
}

func (p *Path) FileExtension() string {
	return filepath.Ext(p.value)
}

func (p *Path) VolumeName() string {
	return filepath.VolumeName(p.value)
}
