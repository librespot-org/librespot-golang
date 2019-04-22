package librespotmobile

import "github.com/librespot-org/librespot-golang/librespot/core"

// MobileSession exposes a simplified subset of the core.Session struct that is compatible with the subset
// of types accepted by gomobile. Most calls are proxied to the underlying core.Session pointer, which we
// cannot expose directly as it uses types incompatible with gomobile.
type MobileSession struct {
	session *core.Session
	player  *MobilePlayer
	mercury *MobileMercury
}

func Login(username string, password string, deviceName string) (*MobileSession, error) {
	sess, err := core.Login(username, password, deviceName)

	if err != nil {
		return nil, err
	}

	return initSessionImpl(sess)
}

func LoginSaved(username string, authData []byte, deviceName string) (*MobileSession, error) {
	sess, err := core.LoginSaved(username, authData, deviceName)

	if err != nil {
		return nil, err
	}

	return initSessionImpl(sess)
}

func initSessionImpl(sess *core.Session) (*MobileSession, error) {
	return &MobileSession{
		session: sess,
		player:  createMobilePlayer(sess),
		mercury: createMobileMercury(sess),
	}, nil
}

func (s *MobileSession) Username() string {
	return s.session.Username()
}

func (s *MobileSession) DeviceId() string {
	return s.session.DeviceId()
}

func (s *MobileSession) ReusableAuthBlob() []byte {
	return s.session.ReusableAuthBlob()
}

func (s *MobileSession) Country() string {
	return s.session.Country()
}

func (s *MobileSession) Player() *MobilePlayer {
	return s.player
}

func (s *MobileSession) Mercury() *MobileMercury {
	return s.mercury
}
