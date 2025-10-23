module static2avif

go 1.25.3

replace pixly/utils => ../utils

require (
	github.com/h2non/filetype v1.1.3
	github.com/karrick/godirwalk v1.17.0
	github.com/panjf2000/ants/v2 v2.11.3
	pixly/utils v0.0.0-00010101000000-000000000000
)

require golang.org/x/sync v0.11.0 // indirect
