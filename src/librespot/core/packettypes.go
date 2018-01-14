package core

const (
	kPacketSecretBlock = 0x02
	kPacketPing        = 0x04
	kPacketStreamChunk = 0x08
	kPacketRequestKey  = 0x0c
	kPacketAesKey      = 0x0d
	kPacketAesKeyError = 0x0e

	kPacketCountryCode = 0x1b

	kPacketPong    = 0x49
	kPacketPongAck = 0x4a
	kPacketPause   = 0x4b

	kPacketProductInfo   = 0x50
	kPacketLegacyWelcome = 0x69

	kPacketLicenseVersion = 0x76

	kPacketAPWelcome   = 0xac
	kPacketAuthFailure = 0xad
)
