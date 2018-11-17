module github.com/librespot-org/librespot-golang/src/librespot

require (
	github.com/badfortrains/mdns v0.0.0-20160325001438-447166384f51
	github.com/davecgh/go-spew v1.1.0 // indirect
	github.com/golang/protobuf v0.0.0-20171113180720-1e59b77b52bf
	github.com/librespot-org/librespot-golang/src/Spotify v0.0.1
	github.com/miekg/dns v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.2.1
	golang.org/x/crypto v0.0.0-20171219041129-d585fd2cc919
	golang.org/x/net v0.0.0-20171212005608-d866cfc389ce // indirect
	golang.org/x/sync v0.0.0-20181108010431-42b317875d0f // indirect
)

replace github.com/librespot-org/librespot-golang/src/Spotify => ../Spotify
