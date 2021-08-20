package check

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	ibcModuleImplementation = "module_ibc.go"
)

// IsIBCModule returns true if the provided module implements the IBC module interface
// we naively check the existence of module_ibc.go for this check
func IsIBCModule(appPath string, moduleName string) (bool, error) {
	absPath, err := filepath.Abs(filepath.Join(appPath, moduleDir, moduleName, ibcModuleImplementation))
	if err != nil {
		return false, err
	}

	_, err = os.Stat(absPath)
	if os.IsNotExist(err) {
		// Not an IBC module
		return false, nil
	}

	return true, err
}

// ForbiddenPacketField returns true if the name is forbidden as a packet name
func ForbiddenPacketField(name string) error {
	switch name {
	case
		"sender",
		"port",
		"channelID":
		return fmt.Errorf("%s is used by the packet scaffolder", name)
	}

	return GoReservedWord(name)
}
