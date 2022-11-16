package wiki

import (
	"github.com/rs/zerolog/log"

	"boomer/internal/myexec"
	"boomer/internal/sdk"
)

func OpenWiki() error {
	sdk.SendEvent(sdk.EventTracking{
		Category: "OpenWiki",
		Action:   "hrp wiki",
	})
	log.Info().Msgf("%s https://httprunner.com", openCmd)
	return myexec.RunCommand(openCmd, "https://httprunner.com")
}
