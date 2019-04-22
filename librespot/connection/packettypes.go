package connection

const (
	PacketSecretBlock    = 0x02
	PacketPing           = 0x04
	PacketStreamChunk    = 0x08
	PacketStreamChunkRes = 0x09
	PacketChannelError   = 0x0a
	PacketChannelAbort   = 0x0b
	PacketRequestKey     = 0x0c
	PacketAesKey         = 0x0d
	PacketAesKeyError    = 0x0e

	PacketImage       = 0x19
	PacketCountryCode = 0x1b

	PacketPong    = 0x49
	PacketPongAck = 0x4a
	PacketPause   = 0x4b

	PacketProductInfo   = 0x50
	PacketLegacyWelcome = 0x69

	PacketLicenseVersion = 0x76

	PacketLogin       = 0xab
	PacketAPWelcome   = 0xac
	PacketAuthFailure = 0xad

	PacketMercuryReq   = 0xb2
	PacketMercurySub   = 0xb3
	PacketMercuryUnsub = 0xb4
)
