[Setup]
AppName=Smart PC Agent
AppVersion={#MyAppVersion}          ; передаётся из CI: iscc /DMyAppVersion=1.2.3
AppPublisher=YourCompany
AppPublisherURL=https://github.com/yourorg/smart-pc
AppSupportURL=https://github.com/yourorg/smart-pc/issues
AppUpdatesURL=https://github.com/yourorg/smart-pc/releases

DefaultDirName={autopf}\SmartPC       ; C:\Program Files\SmartPC
DefaultGroupName=Smart PC Agent
DisableProgramGroupPage=yes           ; не спрашивать про папку в меню Пуск

OutputDir=Output
OutputBaseFilename=SmartPC-Setup-{#MyAppVersion}

SetupIconFile=..\data\assets\icon.ico       ; иконка инсталлятора
WizardStyle=modern                    ; современный вид (vs classic)

Compression=lzma2/ultra64
SolidCompression=yes
PrivilegesRequired=admin              ; нужен UAC для установки в Program Files
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "russian"; MessagesFile: "compiler:Languages\Russian.isl"

[Tasks]
; Галочки которые видит пользователь на шаге "Select Additional Tasks"
Name: "autostart";  Description: "Запускать при старте Windows"; \
  GroupDescription: "Дополнительные параметры:"; Flags: checkedonce
Name: "desktopicon"; Description: "Создать ярлык на рабочем столе"; \
  GroupDescription: "Дополнительные параметры:"; Flags: unchecked

[Files]
; основной бинарь — берётся рядом с installer.iss в CI
Source: "..\dist\smart-pc.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\README.md";          DestDir: "{app}"; Flags: ignoreversion isreadme
Source: "..\LICENSE";            DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\Smart PC Agent";   Filename: "{app}\smart-pc.exe"
Name: "{group}\Удалить Smart PC";  Filename: "{uninstallexe}"

; ярлык на рабочем столе — только если пользователь выбрал галочку
Name: "{autodesktop}\Smart PC Agent"; Filename: "{app}\smart-pc.exe"; \
  Tasks: desktopicon

[Registry]
; добавить в автозапуск если пользователь выбрал галочку
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; \
  ValueType: string; ValueName: "SmartPCAgent"; \
  ValueData: "{app}\smart-pc.exe"; \
  Flags: uninsdeletevalue; Tasks: autostart

; сохранить версию в реестре (для автообновлятора)
Root: HKLM; Subkey: "Software\SmartPC"; \
  ValueType: string; ValueName: "Version"; \
  ValueData: "{#MyAppVersion}"; Flags: uninsdeletekey

[Run]
; запустить агент сразу после установки
Filename: "{app}\smart-pc.exe"; \
  Description: "Запустить Smart PC Agent"; \
  Flags: nowait postinstall skipifsilent

[UninstallRun]
; остановить агент перед удалением
Filename: "taskkill"; Parameters: "/F /IM smart-pc.exe"; \
  Flags: runhidden skipifdoesntexist

[Code]
// проверить что агент не запущен перед установкой (обновление)
function InitializeSetup(): Boolean;
var
  ResultCode: Integer;
begin
  if CheckForMutexes('SmartPCAgentMutex') then begin
    if MsgBox('Smart PC Agent сейчас работает. Остановить для обновления?',
              mbConfirmation, MB_YESNO) = IDYES then begin
      Exec('taskkill', '/F /IM smart-pc.exe', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
      Sleep(1000);
    end else begin
      Result := False;
      Exit;
    end;
  end;
  Result := True;
end;