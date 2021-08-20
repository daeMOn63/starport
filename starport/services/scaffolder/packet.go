package scaffolder

import (
	"fmt"
	"os"

	"github.com/gobuffalo/genny"
	"github.com/tendermint/starport/starport/pkg/check"
	"github.com/tendermint/starport/starport/pkg/field"
	"github.com/tendermint/starport/starport/pkg/gomodulepath"
	"github.com/tendermint/starport/starport/pkg/multiformatname"
	"github.com/tendermint/starport/starport/pkg/placeholder"
	"github.com/tendermint/starport/starport/pkg/xgenny"
	"github.com/tendermint/starport/starport/templates/ibc"
)

// AddPacket adds a new type stype to scaffolded app by using optional type fields.
func (s *Scaffolder) AddPacket(
	tracer *placeholder.Tracer,
	moduleName,
	packetName string,
	packetFields,
	ackFields []string,
	noMessage bool,
) (sm xgenny.SourceModification, err error) {
	path, err := gomodulepath.ParseAt(s.path)
	if err != nil {
		return sm, err
	}

	mfName, err := multiformatname.NewName(moduleName, multiformatname.NoNumber)
	if err != nil {
		return sm, err
	}
	moduleName = mfName.Lowercase

	name, err := multiformatname.NewName(packetName)
	if err != nil {
		return sm, err
	}

	if err := check.ComponentValidity(s.path, moduleName, name, noMessage); err != nil {
		return sm, err
	}

	// Module must implement IBC
	ok, err := check.IsIBCModule(s.path, moduleName)
	if err != nil {
		return sm, err
	}
	if !ok {
		return sm, fmt.Errorf("the module %s doesn't implement IBC module interface", moduleName)
	}

	// Parse packet fields
	parsedPacketFields, err := field.ParseFields(packetFields, check.ForbiddenPacketField)
	if err != nil {
		return sm, err
	}

	// Parse acknowledgment fields
	parsedAcksFields, err := field.ParseFields(ackFields, check.GoReservedWord)
	if err != nil {
		return sm, err
	}

	// Generate the packet
	var (
		g    *genny.Generator
		opts = &ibc.PacketOptions{
			AppName:    path.Package,
			ModulePath: path.RawPath,
			ModuleName: moduleName,
			OwnerName:  owner(path.RawPath),
			PacketName: name,
			Fields:     parsedPacketFields,
			AckFields:  parsedAcksFields,
			NoMessage:  noMessage,
		}
	)
	g, err = ibc.NewPacket(tracer, opts)
	if err != nil {
		return sm, err
	}
	sm, err = xgenny.RunWithValidation(tracer, g)
	if err != nil {
		return sm, err
	}
	pwd, err := os.Getwd()
	if err != nil {
		return sm, err
	}
	return sm, s.finish(pwd, path.RawPath)
}
