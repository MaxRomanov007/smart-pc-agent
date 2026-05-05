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

//go:embed download.ico
var downloadData []byte

func GetDownload() []byte {
	return downloadData
}

//go:embed trash.ico
var trashData []byte

func GetTrash() []byte {
	return trashData
}
