package wolcheck

type InterfaceWoLInfo struct {
	Name    string
	Enabled bool
	Mode    string
}

func GetWoLInfo() ([]InterfaceWoLInfo, error) {
	return getWoLInfo()
}
