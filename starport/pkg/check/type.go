package check

import "fmt"

// ForbiddenTypeField returns true if the name is forbidden as a field name
func ForbiddenTypeField(name string) error {
	switch name {
	case
		"id",
		"appendedValue",
		"creator":
		return fmt.Errorf("%s is used by type scaffolder", name)
	}

	return GoReservedWord(name)
}
