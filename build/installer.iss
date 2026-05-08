[Setup]
AppName=Smart PC Agent
AppVersion={#MyAppVersion}
AppPublisher=MaxRomanov007
AppPublisherURL=https://github.com/MaxRomanov007/smart-pc-agent
AppSupportURL=https://github.com/MaxRomanov007/smart-pc-agent/issues
AppUpdatesURL=https://github.com/MaxRomanov007/smart-pc-agent/releases
DefaultDirName={autopf}\SmartPC
DefaultGroupName=Smart PC Agent
DisableProgramGroupPage=yes
OutputDir=Output
OutputBaseFilename=smart-pc-agent-setup-v{#MyAppVersion}
SetupIconFile=..\data\assets\icon.ico
WizardStyle=modern
Compression=lzma2/ultra64
SolidCompression=yes
PrivilegesRequired=admin
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "russian"; MessagesFile: "compiler:Languages\Russian.isl"

[CustomMessages]
english.TaskAutoStartDesc=Run at Windows startup
russian.TaskAutoStartDesc=Запускать при старте Windows

english.TasksGroupDesc=Additional options:
russian.TasksGroupDesc=Дополнительные параметры:

english.TaskDesktopIconDesc=Create a desktop shortcut
russian.TaskDesktopIconDesc=Создать ярлык на рабочем столе

english.UninstallShortcutName=Uninstall Smart PC
russian.UninstallShortcutName=Удалить Smart PC

english.RunDescription=Launch Smart PC Agent
russian.RunDescription=Запустить Smart PC Agent

english.DeleteInstallerFileDescription=Delete installer file
russian.DeleteInstallerFileDescription=Удалить установщик

english.MsgAppRunning=Smart PC Agent is currently running. Stop it to proceed with the update?
russian.MsgAppRunning=Smart PC Agent сейчас работает. Остановить для обновления?

[Tasks]
Name: "autostart";   Description: "{cm:TaskAutoStartDesc}";   GroupDescription: "{cm:TasksGroupDesc}"; Flags: checkedonce
Name: "desktopicon"; Description: "{cm:TaskDesktopIconDesc}"; GroupDescription: "{cm:TasksGroupDesc}"; Flags: unchecked

[Files]
Source: "..\dist\smart-pc.exe"; DestDir: "{app}"; Flags: ignoreversion
; Source: "..\README.md";         DestDir: "{app}"; Flags: ignoreversion isreadme
Source: "..\LICENSE";           DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\Smart PC Agent";             Filename: "{app}\smart-pc.exe"
Name: "{group}\{cm:UninstallShortcutName}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\Smart PC Agent";       Filename: "{app}\smart-pc.exe"; Tasks: desktopicon

[Registry]
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; \
  ValueType: string; ValueName: "SmartPCAgent"; \
  ValueData: "{app}\smart-pc.exe"; \
  Flags: uninsdeletevalue; Tasks: autostart

Root: HKLM; Subkey: "Software\SmartPC"; \
  ValueType: string; ValueName: "Version"; \
  ValueData: "{#MyAppVersion}"; Flags: uninsdeletekey

[Run]
; Launch the agent after install.
; "skipifsilent" is intentionally absent — silent updates (from the in-app
; updater) must also start the agent automatically after installation.
Filename: "{app}\smart-pc.exe"; \
  Description: "{cm:RunDescription}"; \
  Flags: nowait postinstall

; Delete the installer itself after a successful installation.
; {srcexe} expands to the full path of the running setup executable,
; so this works whether the installer was run manually or dropped in %TEMP%.
Filename: "cmd.exe"; \
  Parameters: "/c ping -n 2 127.0.0.1 >nul & del /f /q ""{srcexe}"""; \
  Description: "{cm:DeleteInstallerFileDescription}"; \
  Flags: runhidden postinstall nowait

[UninstallRun]
Filename: "taskkill"; Parameters: "/F /IM smart-pc.exe"; \
  Flags: runhidden skipifdoesntexist

[UninstallDelete]
Type: filesandordirs; Name: "{localappdata}\smart-pc"