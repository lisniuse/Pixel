package tray

import _ "embed"

//go:embed assets/icon.ico
var icoData []byte

func iconBytes() []byte { return icoData }
