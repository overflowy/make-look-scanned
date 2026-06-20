//go:build !(js && wasm)

// This package is the browser (js/wasm) entrypoint; on every other platform it
// is an empty stub so `go build ./...` and `go vet ./...` stay happy.
package main

func main() {}
