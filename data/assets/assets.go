package assets

import _ "embed"

//go:embed icon.ico
var iconData []byte

func GetIcon() []byte {
	return iconData
}

//go:embed exit.ico
var exitData []byte

func GetExit() []byte {
	return exitData
}

//go:embed house.ico
var houseData []byte

func GetHouse() []byte {
	return houseData
}

//go:embed pc-case.ico
var pcCaseData []byte

func GetPcCase() []byte {
	return pcCaseData
}
