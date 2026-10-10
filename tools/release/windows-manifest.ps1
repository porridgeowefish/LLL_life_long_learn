param(
  [Parameter(Mandatory=$true)][string]$Executable,
  [string]$IconPath = '',
  [string]$Version = '',
  [string]$Description = 'Learn-With-AI'
)
$ErrorActionPreference = 'Stop'
if ([string]::IsNullOrWhiteSpace($IconPath)) { $IconPath = Join-Path $PSScriptRoot '..\..\scripts\assets\learn-with-ai.ico' }
if ([string]::IsNullOrWhiteSpace($Version)) { $Version = (Get-Content -LiteralPath (Join-Path $PSScriptRoot '..\..\package.json') -Raw | ConvertFrom-Json).version }
if (-not (Test-Path -LiteralPath $IconPath -PathType Leaf)) { throw "Missing application icon: $IconPath" }
Add-Type -TypeDefinition @'
using System;
using System.IO;
using System.Text;
using System.Collections.Generic;
using System.Runtime.InteropServices;
public static class LearnInstallerResource {
  [DllImport("kernel32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
  public static extern IntPtr BeginUpdateResource(string path, bool delete);
  [DllImport("kernel32.dll", SetLastError=true)]
  public static extern bool UpdateResource(IntPtr handle, IntPtr type, IntPtr name, ushort language, byte[] data, uint size);
  [DllImport("kernel32.dll", SetLastError=true)]
  public static extern bool EndUpdateResource(IntPtr handle, bool discard);
  static void Put(IntPtr handle,int type,int id,byte[] bytes) {
    if(!UpdateResource(handle,(IntPtr)type,(IntPtr)id,1033,bytes,(uint)bytes.Length))throw new System.ComponentModel.Win32Exception(Marshal.GetLastWin32Error());
  }
  public static void Icons(IntPtr handle,byte[] ico) {
    using(var reader=new BinaryReader(new MemoryStream(ico))) {
      if(reader.ReadUInt16()!=0||reader.ReadUInt16()!=1)throw new InvalidDataException("Invalid ICO header");
      int count=reader.ReadUInt16();if(count<1||count>32)throw new InvalidDataException("Invalid ICO count");
      using(var group=new MemoryStream())using(var writer=new BinaryWriter(group)){
        writer.Write((ushort)0);writer.Write((ushort)1);writer.Write((ushort)count);
        for(int i=0;i<count;i++) {
          byte[] entry=reader.ReadBytes(16);if(entry.Length!=16)throw new InvalidDataException("Truncated ICO");
          int length=BitConverter.ToInt32(entry,8),offset=BitConverter.ToInt32(entry,12);
          if(length<1||offset<6+count*16||offset>ico.Length-length)throw new InvalidDataException("Invalid ICO frame");
          byte[] data=new byte[length];Buffer.BlockCopy(ico,offset,data,0,length);
          Put(handle,3,i+1,data);writer.Write(entry,0,12);writer.Write((ushort)(i+1));
        }
        Put(handle,14,1,group.ToArray());
      }
    }
  }
  static void Align(BinaryWriter writer) {while(writer.BaseStream.Position%4!=0)writer.Write((byte)0);}
  static byte[] Node(string key,byte[] value,ushort type,params byte[][] children) {
    using(var buffer=new MemoryStream())using(var writer=new BinaryWriter(buffer)){
      writer.Write((ushort)0);writer.Write((ushort)(type==1?value.Length/2:value.Length));writer.Write(type);
      writer.Write(Encoding.Unicode.GetBytes(key+"\0"));Align(writer);writer.Write(value);
      foreach(var child in children){Align(writer);writer.Write(child);}
      ushort length=checked((ushort)buffer.Length);buffer.Position=0;writer.Write(length);return buffer.ToArray();
    }
  }
  static byte[] Text(string key,string value){return Node(key,Encoding.Unicode.GetBytes(value+"\0"),1);}
  public static void Version(IntPtr handle,string version,string description) {
    string numeric=version.Split('-')[0];string[] parts=numeric.Split('.');
    if(parts.Length!=3)throw new InvalidDataException("Expected major.minor.patch version");
    uint major=ushort.Parse(parts[0]),minor=ushort.Parse(parts[1]),patch=ushort.Parse(parts[2]);
    byte[] fixedInfo;
    using(var buffer=new MemoryStream())using(var w=new BinaryWriter(buffer)){
      foreach(uint value in new uint[]{0xFEEF04BD,0x00010000,(major<<16)|minor,patch<<16,(major<<16)|minor,patch<<16,0x3F,0,0x00040004,1,0,0,0})w.Write(value);
      fixedInfo=buffer.ToArray();
    }
    byte[] strings=Node("StringFileInfo",new byte[0],1,Node("040904b0",new byte[0],1,
      Text("CompanyName","Learn-With-AI"),Text("FileDescription",description),Text("FileVersion",version),
      Text("InternalName","learn-with-ai"),Text("LegalCopyright","Copyright Learn-With-AI contributors"),
      Text("OriginalFilename","learn-with-ai.exe"),Text("ProductName","Learn-With-AI"),Text("ProductVersion",version)));
    byte[] translation=Node("VarFileInfo",new byte[0],1,Node("Translation",new byte[]{9,4,0xB0,4},0));
    Put(handle,16,1,Node("VS_VERSION_INFO",fixedInfo,0,strings,translation));
  }
}
'@
$manifest = '<?xml version="1.0" encoding="UTF-8" standalone="yes"?><assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0"><assemblyIdentity version="1.0.0.0" processorArchitecture="*" name="learn-with-ai.desktop" type="win32"/><trustInfo xmlns="urn:schemas-microsoft-com:asm.v3"><security><requestedPrivileges><requestedExecutionLevel level="asInvoker" uiAccess="false"/></requestedPrivileges></security></trustInfo><compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1"><application><supportedOS Id="{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}"/></application></compatibility><application xmlns="urn:schemas-microsoft-com:asm.v3"><windowsSettings><dpiAwareness xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">PerMonitorV2</dpiAwareness><longPathAware xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">true</longPathAware></windowsSettings></application></assembly>'
$bytes = [Text.Encoding]::UTF8.GetBytes($manifest)
$handle = [LearnInstallerResource]::BeginUpdateResource($Executable, $false)
if ($handle -eq [IntPtr]::Zero) { throw "BeginUpdateResource failed: $([Runtime.InteropServices.Marshal]::GetLastWin32Error())" }
try {
  if (-not [LearnInstallerResource]::UpdateResource($handle, [IntPtr]24, [IntPtr]1, 1033, $bytes, $bytes.Length)) { throw "UpdateResource failed: $([Runtime.InteropServices.Marshal]::GetLastWin32Error())" }
  [LearnInstallerResource]::Icons($handle, [IO.File]::ReadAllBytes([IO.Path]::GetFullPath($IconPath)))
  [LearnInstallerResource]::Version($handle, $Version, $Description)
  if (-not [LearnInstallerResource]::EndUpdateResource($handle, $false)) { throw "EndUpdateResource failed: $([Runtime.InteropServices.Marshal]::GetLastWin32Error())" }
  $handle = [IntPtr]::Zero
} finally {
  if ($handle -ne [IntPtr]::Zero) { [LearnInstallerResource]::EndUpdateResource($handle, $true) | Out-Null }
}
