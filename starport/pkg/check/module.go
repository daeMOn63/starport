package check

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	appanalysis "github.com/tendermint/starport/starport/pkg/cosmosanalysis/app"
	"github.com/tendermint/starport/starport/templates/module"
	modulecreate "github.com/tendermint/starport/starport/templates/module/create"
)

const (
	appPkg    = "app"
	moduleDir = "x"
)

func ModuleExists(appPath string, moduleName string) (bool, error) {
	absPath, err := filepath.Abs(filepath.Join(appPath, moduleDir, moduleName))
	if err != nil {
		return false, err
	}

	_, err = os.Stat(absPath)
	if os.IsNotExist(err) {
		// The module doesn't exist
		return false, nil
	}

	return true, err
}

func ModuleName(moduleName string) error {
	// go keyword
	if token.Lookup(moduleName).IsKeyword() {
		return fmt.Errorf("%s is a Go keyword", moduleName)
	}

	// name of default registered module
	switch moduleName {
	case
		"auth",
		"genutil",
		"bank",
		"capability",
		"staking",
		"mint",
		"distr",
		"gov",
		"params",
		"crisis",
		"slashing",
		"ibc",
		"upgrade",
		"evidence",
		"transfer",
		"vesting":
		return fmt.Errorf("%s is a default module", moduleName)
	}
	return nil
}

func IsWasmImported(appPath, wasmImport string) (bool, error) {
	abspath, err := filepath.Abs(filepath.Join(appPath, appPkg))
	if err != nil {
		return false, err
	}
	fset := token.NewFileSet()
	all, err := parser.ParseDir(fset, abspath, func(os.FileInfo) bool { return true }, parser.ImportsOnly)
	if err != nil {
		return false, err
	}
	for _, pkg := range all {
		for _, f := range pkg.Files {
			for _, imp := range f.Imports {
				if strings.Contains(imp.Path.Value, wasmImport) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// Dependencies perform checks on the dependencies
func Dependencies(dependencies []modulecreate.Dependency) error {
	depMap := make(map[string]struct{})
	for _, dep := range dependencies {
		// check the dependency has been registered
		if err := appanalysis.CheckKeeper(module.PathAppModule, dep.KeeperName); err != nil {
			return fmt.Errorf(
				"the module cannot have %s as a dependency: %s",
				dep.Name,
				err.Error(),
			)
		}

		// check duplicated
		_, ok := depMap[dep.Name]
		if ok {
			return fmt.Errorf("%s is a duplicated dependency", dep)
		}
		depMap[dep.Name] = struct{}{}
	}

	return nil
}
