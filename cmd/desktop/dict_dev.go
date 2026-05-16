//go:build !production

package main

func resolveDictPath() string {
	return "../../dict.json"
}
