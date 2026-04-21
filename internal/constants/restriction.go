package constants

import (
	"fmt"

	gamedesign "github.com/kenyamaneko/overload-party-common/packages/game-design-constants"
)

// RestrictionCopyCount returns the maximum number of copies allowed in a deck
// for the given restriction value. Returns error for unknown values instead of
// silently assuming unlimited — an unknown restriction here means card_definitions
// has invalid data and callers must surface it, not absorb it.
func RestrictionCopyCount(restriction string) (int, error) {
	switch restriction {
	case gamedesign.RestrictionForbidden:
		return 0, nil
	case gamedesign.RestrictionLimited:
		return 1, nil
	case gamedesign.RestrictionSemiLimited:
		return 2, nil
	case gamedesign.RestrictionUnlimited:
		return 3, nil
	}
	return 0, fmt.Errorf("unknown restriction %q", restriction)
}
