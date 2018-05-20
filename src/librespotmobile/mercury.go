package librespotmobile

import (
	"librespot/core"
	"librespot/mercury"
)

type MobileMercury struct {
	mercury *mercury.Client
}

func createMobileMercury(session *core.Session) *MobileMercury {
	return &MobileMercury{
		mercury: session.Mercury(),
	}
}
