package scaffolder

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/gobuffalo/genny"
	"github.com/tendermint/starport/starport/pkg/check"
	"github.com/tendermint/starport/starport/pkg/cmdrunner"
	"github.com/tendermint/starport/starport/pkg/cmdrunner/step"
	"github.com/tendermint/starport/starport/pkg/cosmosver"
	"github.com/tendermint/starport/starport/pkg/gocmd"
	"github.com/tendermint/starport/starport/pkg/gomodulepath"
	"github.com/tendermint/starport/starport/pkg/multiformatname"
	"github.com/tendermint/starport/starport/pkg/placeholder"
	"github.com/tendermint/starport/starport/pkg/validation"
	"github.com/tendermint/starport/starport/pkg/xgenny"
	modulecreate "github.com/tendermint/starport/starport/templates/module/create"
	moduleimport "github.com/tendermint/starport/starport/templates/module/import"
)

const (
	wasmImport    = "github.com/CosmWasm/wasmd"
	wasmVersion   = "v0.16.0"
	extrasImport  = "github.com/tendermint/spm-extras"
	extrasVersion = "v0.1.0"
)

// moduleCreationOptions holds options for creating a new module
type moduleCreationOptions struct {
	// chainID is the chain's id.
	ibc bool

	// homePath of the chain's config dir.
	ibcChannelOrdering string

	// list of module depencies
	dependencies []modulecreate.Dependency
}

// ModuleCreationOption configures Chain.
type ModuleCreationOption func(*moduleCreationOptions)

// WithIBC scaffolds a module with IBC enabled
func WithIBC() ModuleCreationOption {
	return func(m *moduleCreationOptions) {
		m.ibc = true
	}
}

// WithIBCChannelOrdering configures channel ordering of the IBC module
func WithIBCChannelOrdering(ordering string) ModuleCreationOption {
	return func(m *moduleCreationOptions) {
		switch ordering {
		case "ordered":
			m.ibcChannelOrdering = "ORDERED"
		case "unordered":
			m.ibcChannelOrdering = "UNORDERED"
		default:
			m.ibcChannelOrdering = "NONE"
		}
	}
}

// WithDependencies specifies the name of the modules that the module depends on
func WithDependencies(dependencies []modulecreate.Dependency) ModuleCreationOption {
	return func(m *moduleCreationOptions) {
		m.dependencies = dependencies
	}
}

// CreateModule creates a new empty module in the scaffolded app
func (s *Scaffolder) CreateModule(
	tracer *placeholder.Tracer,
	moduleName string,
	options ...ModuleCreationOption,
) (sm xgenny.SourceModification, err error) {
	mfName, err := multiformatname.NewName(moduleName, multiformatname.NoNumber)
	if err != nil {
		return sm, err
	}
	moduleName = mfName.Lowercase

	// Check if the module name is valid
	if err := check.ModuleName(moduleName); err != nil {
		return sm, err
	}

	// Check if the module already exist
	ok, err := check.ModuleExists(s.path, moduleName)
	if err != nil {
		return sm, err
	}
	if ok {
		return sm, fmt.Errorf("the module %v already exists", moduleName)
	}

	// Apply the options
	var creationOpts moduleCreationOptions
	for _, apply := range options {
		apply(&creationOpts)
	}

	// Check dependencies
	if err := check.Dependencies(creationOpts.dependencies); err != nil {
		return sm, err
	}

	path, err := gomodulepath.ParseAt(s.path)
	if err != nil {
		return sm, err
	}
	opts := &modulecreate.CreateOptions{
		ModuleName:   moduleName,
		ModulePath:   path.RawPath,
		AppName:      path.Package,
		OwnerName:    owner(path.RawPath),
		IsIBC:        creationOpts.ibc,
		IBCOrdering:  creationOpts.ibcChannelOrdering,
		Dependencies: creationOpts.dependencies,
	}

	// Generator from Cosmos SDK version
	g, err := modulecreate.NewStargate(opts)
	if err != nil {
		return sm, err
	}
	gens := []*genny.Generator{g}

	// Scaffold IBC module
	if opts.IsIBC {
		g, err = modulecreate.NewIBC(tracer, opts)
		if err != nil {
			return sm, err
		}
		gens = append(gens, g)
	}
	sm, err = xgenny.RunWithValidation(tracer, gens...)
	if err != nil {
		return sm, err
	}

	// Modify app.go to register the module
	newSourceModification, runErr := xgenny.RunWithValidation(tracer, modulecreate.NewStargateAppModify(tracer, opts))
	sm.Merge(newSourceModification)
	var validationErr validation.Error
	if runErr != nil && !errors.As(runErr, &validationErr) {
		return sm, runErr
	}

	// Generate proto and format the source
	pwd, err := os.Getwd()
	if err != nil {
		return sm, err
	}
	if err := s.finish(pwd, path.RawPath); err != nil {
		return sm, err
	}
	return sm, runErr
}

// ImportModule imports specified module with name to the scaffolded app.
func (s *Scaffolder) ImportModule(tracer *placeholder.Tracer, name string) (sm xgenny.SourceModification, err error) {
	// Only wasm is currently supported
	if name != "wasm" {
		return sm, errors.New("module cannot be imported. Supported module: wasm")
	}

	ok, err := check.IsWasmImported(s.path, wasmImport)
	if err != nil {
		return sm, err
	}
	if ok {
		return sm, errors.New("wasm is already imported")
	}

	path, err := gomodulepath.ParseAt(s.path)
	if err != nil {
		return sm, err
	}

	// run generator
	g, err := moduleimport.NewStargate(tracer, &moduleimport.ImportOptions{
		Feature:          name,
		AppName:          path.Package,
		BinaryNamePrefix: path.Root,
	})
	if err != nil {
		return sm, err
	}

	sm, err = xgenny.RunWithValidation(tracer, g)
	if err != nil {
		var validationErr validation.Error
		if errors.As(err, &validationErr) {
			// TODO: implement a more generic method when there will be new methods to import wasm
			return sm, errors.New("wasm cannot be imported. Apps initialized with Starport <=0.16.2 must downgrade Starport to 0.16.2 to import wasm")
		}
		return sm, err
	}

	// import a specific version of ComsWasm
	// NOTE(dshulyak) it must be installed after validation
	if err := s.installWasm(); err != nil {
		return sm, err
	}

	pwd, err := os.Getwd()
	if err != nil {
		return sm, err
	}
	return sm, s.finish(pwd, path.RawPath)
}

func (s *Scaffolder) installWasm() error {
	switch s.version {
	case cosmosver.StargateZeroFourtyAndAbove:
		return cmdrunner.
			New().
			Run(context.Background(),
				step.New(step.Exec(gocmd.Name(), "get", gocmd.PackageLiteral(wasmImport, wasmVersion))),
				step.New(step.Exec(gocmd.Name(), "get", gocmd.PackageLiteral(extrasImport, extrasVersion))),
			)
	default:
		return errors.New("version not supported")
	}
}
