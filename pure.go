package pure

import (
	"os"
	"path/filepath"
	"regexp"
)

func QuantityValue(quant string) string {
	reg := regexp.MustCompile(`(\d+(\.\d+)?)`)
	return reg.FindString(quant)
}

func QuantityUnit(quant string) string {
	reg := regexp.MustCompile("([a-zA-Z_-]+[@#%/^.0-9]*)+")
	return reg.FindString(quant)
}

func PathDirectory(path string) string {
	return filepath.Dir(path)
}

func PathBase(path string) string {
	return filepath.Base(path)
}

func PathFileExtension(path string) string {
	return filepath.Ext(path)
}

func PathVolumeName(path string) string {
	return filepath.VolumeName(path)
}

func EnvExpand(env string) string {
	return os.ExpandEnv(env)
}
