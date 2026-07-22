package main

import "runtime/debug"

// Injected by GoReleaser via ldflags on release builds
var version = "dev"

func Version() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return version
}
