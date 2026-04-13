package constants

import (
	gamedesign "github.com/kenyamaneko/overload-party-common/packages/game-design-constants"
)

// RestrictionCopyCount returns the maximum number of copies allowed in a deck.
func RestrictionCopyCount(restriction string) int {
	switch restriction {
	case gamedesign.RestrictionForbidden:
		return 0
	case gamedesign.RestrictionLimited:
		return 1
	case gamedesign.RestrictionSemiLimited:
		return 2
	default:
		return 3
	}
}
