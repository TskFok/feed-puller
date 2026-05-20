package paths

import (
	"path/filepath"
	"strings"
)

// Mapper 将 aria2 / 数据库中的宿主机路径前缀映射为容器内可访问的路径。
type Mapper struct {
	hostPrefix      string
	containerPrefix string
}

// NewMapper 创建路径映射；任一前缀为空时 Map 为恒等变换。
func NewMapper(hostPrefix, containerPrefix string) Mapper {
	hostPrefix = cleanPrefix(hostPrefix)
	containerPrefix = cleanPrefix(containerPrefix)
	return Mapper{hostPrefix: hostPrefix, containerPrefix: containerPrefix}
}

func cleanPrefix(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = filepath.Clean(p)
	if p == "/" {
		return p
	}
	return strings.TrimRight(p, string(filepath.Separator))
}

// Map 将 path 映射到容器内路径（若配置了前缀且 path 以 hostPrefix 开头）。
func (m Mapper) Map(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || m.hostPrefix == "" || m.containerPrefix == "" {
		return path
	}
	if path == m.hostPrefix {
		return m.containerPrefix
	}
	sep := string(filepath.Separator)
	if strings.HasPrefix(path, m.hostPrefix+sep) {
		rest := strings.TrimPrefix(path, m.hostPrefix)
		return m.containerPrefix + rest
	}
	return path
}

// Enabled 是否已配置路径映射。
func (m Mapper) Enabled() bool {
	return m.hostPrefix != "" && m.containerPrefix != ""
}
