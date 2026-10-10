module github.com/xmz14/lll

go 1.26.0

replace github.com/giraffesyo/pdf => ./third_party/giraffesyo-pdf

require github.com/fsnotify/fsnotify v1.10.1

require (
	github.com/jchv/go-webview2 v0.0.0-20260205173254-56598839c808 // indirect
	github.com/jchv/go-winloader v0.0.0-20250406163304-c1995be93bd1 // indirect
)

require (
	github.com/giraffesyo/pdf v0.7.0
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.30.0
)
