; Inno Setup script for the Windows installer. tools/package_installers.py
; passes the version and directories in, so this file holds no release state.
#ifndef MyAppVersion
  #define MyAppVersion "1.0.0"
#endif
#ifndef SourceRoot
  #define SourceRoot "bin"
#endif
#ifndef OutputRoot
  #define OutputRoot "release"
#endif

#define MyAppName "GuildSync Voice Mute"
#define MyAppExeName "GuildSync-Voice-Mute.exe"
#define MyAppPublisher "GuildSync"
#define MyAppURL "https://github.com/skperdue68/GuildSync-Voice-Mute"

[Setup]
AppId={{6F1A9C24-3B7D-4E58-9C0A-2D5E8B71C3F4}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}/releases
DefaultDirName={autopf}\{#MyAppName}
DisableProgramGroupPage=yes
DefaultGroupName={#MyAppName}
OutputDir={#OutputRoot}
OutputBaseFilename=GuildSync-Voice-Mute-{#MyAppVersion}-Windows-Setup
Compression=lzma
SolidCompression=yes
WizardStyle=modern
; Per-user install, so neither the installer nor the app needs elevation.
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
UninstallDisplayIcon={app}\{#MyAppExeName}
VersionInfoVersion={#MyAppVersion}
VersionInfoCompany={#MyAppPublisher}
VersionInfoDescription={#MyAppName} installer
VersionInfoProductName={#MyAppName}
VersionInfoProductVersion={#MyAppVersion}

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Additional shortcuts:"; Flags: checkedonce

[Files]
; WebView2 is not bundled: Windows 10 and 11 install the Evergreen runtime.
Source: "{#SourceRoot}\{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "Launch {#MyAppName}"; Flags: nowait postinstall skipifsilent
