package i18n

// Tray items

func MsgOpenDashboard() string   { return T("OpenDashboard", "Open dashboard") }
func MsgOpenInterface() string   { return T("OpenInterface", "Open interface") }
func MsgQuit() string            { return T("Quit", "Quit") }
func MsgQuitTooltip() string     { return T("QuitTooltip", "Quit Smart PC agent") }
func MsgTrayTitle() string       { return T("TrayTitle", "Smart PC") }
func MsgTrayTooltip() string     { return T("TrayTooltip", "Smart PC Control Agent") }
func MsgCheckForUpdates() string { return T("CheckForUpdates", "Check for updates") }
func MsgChecking() string        { return T("Checking", "Checking…") }
func MsgUpdating() string        { return T("Updating", "Updating…") }
func MsgAuthorizeWaker() string  { return T("AuthorizeWaker", "Authorize waker") }
func MsgInstallWaker() string    { return T("InstallWaker", "Install waker") }
func MsgUninstallWaker() string  { return T("UninstallWaker", "Uninstall waker") }

func MsgInstallWakerSSHUnavailable() string {
	return T("InstallWakerSSHUnavailable", "Install waker (SSH unavailable)")
}

func MsgUninstallWakerSSHUnavailable() string {
	return T("UninstallWakerSSHUnavailable", "Uninstall waker (SSH unavailable)")
}

func MsgUpdateTo(version string) string {
	return TData("UpdateTo", "Update to {{.Version}}", map[string]any{"Version": version})
}

// Tooltips

func MsgAuthorizeWakerTooltip() string {
	return T("AuthorizeWakerTooltip", "Allow the waker to receive commands")
}

func MsgInstallWakerTooltip() string {
	return T("InstallWakerTooltip", "Install the remote waker agent via SSH")
}

func MsgUninstallWakerTooltip() string {
	return T("UninstallWakerTooltip", "Remove the remote waker agent via SSH")
}

func MsgInstallWakerSSHUnavailableTooltip() string {
	return T("InstallWakerSSHUnavailableTooltip", "The device at the waker IP does not support SSH")
}

func MsgUninstallWakerSSHUnavailableTooltip() string {
	return T(
		"UninstallWakerSSHUnavailableTooltip",
		"The device at the waker IP does not support SSH",
	)
}

// Zenity dialogs

func MsgAlreadyUpToDate() string   { return T("AlreadyUpToDate", "You are running the latest version.") }
func MsgUpdateFailed() string      { return T("UpdateFailed", "Update failed") }
func MsgInstallingWaker() string   { return T("InstallingWaker", "Installing waker") }
func MsgUninstallingWaker() string { return T("UninstallingWaker", "Uninstalling waker") }

func MsgClickOKToAuthorize() string { return T("ClickOKToAuthorize", "Click OK to authorize waker") }
func MsgWakerUninstalledSuccessfully() string {
	return T("WakerUninstalledSuccessfully", "Waker uninstalled successfully.")
}

func MsgFailedToInstallWaker() string {
	return T("FailedToInstallWaker", "Failed to install waker. See the logs for details.")
}

func MsgFailedToUninstallWaker() string {
	return T("FailedToUninstallWaker", "Failed to uninstall waker. See the logs for details.")
}

func MsgSSHCredentialsInstall() string {
	return T("SSHCredentialsInstall", "Type SSH credentials to install waker")
}

func MsgSSHCredentialsUninstall() string {
	return T("SSHCredentialsUninstall", "Type SSH credentials to uninstall waker")
}
func MsgUpdateAvailableTitle() string { return T("UpdateAvailableTitle", "Update available") }
func MsgUpdateOKLabel() string        { return T("UpdateOKLabel", "Update") }
func MsgCancelLabel() string          { return T("CancelLabel", "Cancel") }

func MsgUpdateAvailableNotify(version string) string {
	return TData("UpdateAvailableNotify",
		"Update {{.Version}} is available. Click the tray icon to install.",
		map[string]any{"Version": version})
}

func MsgUpdateAvailableQuestion(version string) string {
	return TData("UpdateAvailableQuestion", "Update agent to {{.Version}}?",
		map[string]any{"Version": version})
}

func MsgCouldNotReachUpdateServer(errMsg string) string {
	return TData("CouldNotReachUpdateServer",
		"Could not reach update server:\n{{.Error}}",
		map[string]any{"Error": errMsg})
}

// Install steps

func MsgStepDialSSH() string        { return T("StepDialSSH", "Creating an SSH connection") }
func MsgStepDetectArch() string     { return T("StepDetectArch", "Detecting architecture") }
func MsgStepGetLatestTag() string   { return T("StepGetLatestTag", "Getting latest version") }
func MsgStepDownloadBinary() string { return T("StepDownloadBinary", "Downloading binary") }
func MsgStepUpload() string         { return T("StepUpload", "Uploading file") }

func MsgStepConfigureAutostart() string { return T("StepConfigureAutostart", "Configuring autostart") }

func MsgStepRemoveTempFile() string  { return T("StepRemoveTempFile", "Removing temporary file") }
func MsgStepCloseConnection() string { return T("StepCloseConnection", "Closing connection") }

func MsgStepUnregisterAllPcs() string { return T("StepUnregisterAllPcs", "Unregistering all pcs") }
func MsgStepRemoveAutostart() string  { return T("StepRemoveAutostart", "Removing Autostart") }
func MsgStepRemoveBinary() string     { return T("StepRemoveBinary", "Removing Binary") }
func MsgStepRemoveCache() string      { return T("StepRemoveCache", "Removing Cache") }
