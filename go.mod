module github.com/SakurakaiCat/x365-mobile

go 1.26.0

require github.com/SakurakaiCat/x365-core v0.0.0

require (
	github.com/andybalholm/brotli v1.0.6 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/refraction-networking/utls v1.8.3-0.20260301010127-aa6edf4b11af // indirect
	golang.org/x/crypto v0.55.0 // indirect
	golang.org/x/mobile v0.0.0-20260821190718-4776eadac327 // indirect
	golang.org/x/mod v0.39.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/tools v0.49.0 // indirect
)

replace github.com/SakurakaiCat/x365-core => ../x365-core

tool golang.org/x/mobile/cmd/gobind
