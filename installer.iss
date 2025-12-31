; Inno Setup Script for Unreal Free Assets Monitor
; Beautiful modern installer with Unreal Engine styling

#define MyAppName "Unreal Free Assets Monitor"
#define MyAppVersion "1.0.1"
#define MyAppPublisher "Christer Söderlund"
#define MyAppExeName "unreal-free-assets.exe"
#define MyAppAssocName MyAppName + " File"
#define MyAppAssocExt ".ufa"
#define MyAppAssocKey StringChange(MyAppAssocName, " ", "") + MyAppAssocExt

[Setup]
AppId={{8F3E9A7B-4C2D-4E5F-A1B8-9D6C3E2F1A0B}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
AllowNoIcons=yes
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog
OutputDir=dist
OutputBaseFilename=UnrealFreeAssetsSetup-{#MyAppVersion}
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern
WizardSizePercent=120
WizardResizable=no

; Modern styling
SetupIconFile=icon.ico
UninstallDisplayIcon={app}\{#MyAppExeName}
DisableWelcomePage=no
DisableDirPage=no
DisableProgramGroupPage=yes

; Version info
VersionInfoVersion={#MyAppVersion}
VersionInfoCompany={#MyAppPublisher}
VersionInfoDescription=Monitor FAB & Unreal Marketplace for free assets
VersionInfoTextVersion={#MyAppVersion}
VersionInfoProductName={#MyAppName}
VersionInfoProductVersion={#MyAppVersion}

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Messages]
WelcomeLabel1=Welcome to [name] Setup
WelcomeLabel2=This will install [name/ver] on your computer.%n%nThe app monitors FAB and Unreal Marketplace for free asset packs, notifying you when new ones become available.%n%nFeatures:%n  • System tray application%n  • Hourly automatic checks%n  • Beautiful native UI for viewing assets%n  • Windows notifications for new finds

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked
Name: "startupicon"; Description: "Launch at Windows startup (recommended)"; GroupDescription: "Startup options:"; Flags: checkedonce

[Files]
Source: "unreal-free-assets.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "icon.ico"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; IconFilename: "{app}\icon.ico"; Comment: "Monitor for free Unreal assets"
Name: "{group}\Uninstall {#MyAppName}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; IconFilename: "{app}\icon.ico"; Tasks: desktopicon; Comment: "Monitor for free Unreal assets"
Name: "{userstartup}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; IconFilename: "{app}\icon.ico"; Tasks: startupicon

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "Launch {#MyAppName}"; Flags: nowait postinstall skipifsilent

[UninstallRun]
Filename: "taskkill"; Parameters: "/F /IM {#MyAppExeName}"; Flags: runhidden; RunOnceId: "KillApp"

[UninstallDelete]
Type: filesandordirs; Name: "{userappdata}\UnrealFreeAssets"

[Code]
var
  DownloadPage: TDownloadWizardPage;

function InitializeSetup(): Boolean;
var
  ResultCode: Integer;
begin
  // Kill any running instance
  Exec('taskkill', '/F /IM unreal-free-assets.exe', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Result := True;
end;

procedure InitializeWizard;
begin
  // Custom welcome styling would go here
  WizardForm.WelcomeLabel1.Font.Size := 14;
  WizardForm.WelcomeLabel1.Font.Style := [fsBold];
end;

function NextButtonClick(CurPageID: Integer): Boolean;
begin
  Result := True;
end;

function UpdateReadyMemo(Space, NewLine, MemoUserInfoInfo, MemoDirInfo, MemoTypeInfo, MemoComponentsInfo, MemoGroupInfo, MemoTasksInfo: String): String;
var
  S: String;
begin
  S := '';
  S := S + 'Installation Summary' + NewLine + NewLine;
  
  if MemoDirInfo <> '' then
    S := S + MemoDirInfo + NewLine + NewLine;
    
  if MemoTasksInfo <> '' then
    S := S + MemoTasksInfo + NewLine + NewLine;
  
  S := S + 'The application will:' + NewLine;
  S := S + Space + '• Run in your system tray' + NewLine;
  S := S + Space + '• Check for free assets every hour' + NewLine;
  S := S + Space + '• Notify you of new finds' + NewLine;
  
  Result := S;
end;
