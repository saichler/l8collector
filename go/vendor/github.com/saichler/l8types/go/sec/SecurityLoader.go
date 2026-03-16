package sec

import (
	"bytes"
	"errors"
	"github.com/saichler/l8types/go/ifs"
	"os"
	"plugin"
)

// LoadSecurityProvider loads the security provider plugin from /var/loader.so.
// This function uses Go's plugin system to dynamically load the security implementation.
func LoadSecurityProvider(args ...interface{}) (ifs.ISecurityProvider, error) {
	loaderFile, err := plugin.Open("/var/loader.so")
	sp := NewShallowSecurityProvider()
	if err != nil {
		return sp, errors.New("Failed to load security provider #1: " + err.Error())
	}
	loader, err := loaderFile.Lookup("Loader")
	if err != nil {
		return sp, errors.New("Failed to load security provider #2: " + err.Error())
	}
	if loader == nil {
		return sp, errors.New("Failed to load security provider #3: Nil Loader")
	}
	loaderInterface := *loader.(*ifs.ISecurityProviderLoader)
	securityLoader := loaderInterface.(ifs.ISecurityProviderLoader).(ifs.ISecurityProviderLoader)
	return securityLoader.LoadSecurityProvider(args...)
}

func SeekResource(path string, filename string) string {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return ""
	}
	if fileInfo.Name() == filename {
		return path
	}
	if fileInfo.IsDir() {
		files, err := os.ReadDir(path)
		if err != nil {
			return ""
		}
		for _, file := range files {
			found := SeekResource(pathOf(path, file), filename)
			if found != "" {
				return found
			}
		}
	}
	return ""
}

func pathOf(path string, file os.DirEntry) string {
	buff := bytes.Buffer{}
	buff.WriteString(path)
	buff.WriteString("/")
	buff.WriteString(file.Name())
	return buff.String()
}
