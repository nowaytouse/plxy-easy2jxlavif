module dynamic2h266mov

go 1.25.3

require (
	github.com/karrick/godirwalk v1.17.0
	pixly/utils v0.0.0-00010101000000-000000000000
)

require (
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/h2non/filetype v1.1.3 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	golang.org/x/sys v0.15.0 // indirect
)

replace (
	github.com/pixly/archive/shared => ../shared
	pixly/utils => ../utils
)
