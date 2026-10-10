param([Parameter(Mandatory=$true)][string]$Executable)
$ErrorActionPreference='Stop'
Add-Type -TypeDefinition @'
using System;using System.Runtime.InteropServices;
public static class AppResourceReader {
 [DllImport("kernel32.dll",CharSet=CharSet.Unicode)]public static extern IntPtr LoadLibraryEx(string p,IntPtr file,uint flags);
 [DllImport("kernel32.dll")]public static extern IntPtr FindResource(IntPtr h,IntPtr name,IntPtr type);
 [DllImport("kernel32.dll")]public static extern IntPtr LoadResource(IntPtr h,IntPtr resource);
 [DllImport("kernel32.dll")]public static extern IntPtr LockResource(IntPtr resource);
 [DllImport("kernel32.dll")]public static extern uint SizeofResource(IntPtr h,IntPtr resource);
 [DllImport("kernel32.dll")]public static extern bool FreeLibrary(IntPtr h);
 public static byte[] Read(string p,int type) {
  IntPtr h=LoadLibraryEx(p,IntPtr.Zero,2);if(h==IntPtr.Zero)throw new Exception("Cannot load PE resources");
  try{IntPtr r=FindResource(h,(IntPtr)1,(IntPtr)type);if(r==IntPtr.Zero)throw new Exception("Missing resource "+type);
   byte[] data=new byte[SizeofResource(h,r)];Marshal.Copy(LockResource(LoadResource(h,r)),data,0,data.Length);return data;
  }finally{FreeLibrary(h);}
 }
}
'@
$group=[AppResourceReader]::Read($Executable,14)
$version=[Diagnostics.FileVersionInfo]::GetVersionInfo($Executable)
$manifest=[Text.Encoding]::UTF8.GetString([AppResourceReader]::Read($Executable,24))
$firstFrame=[AppResourceReader]::Read($Executable,3)
@{version=$version.ProductVersion;product=$version.ProductName;description=$version.FileDescription;count=[BitConverter]::ToUInt16($group,4);lastWidth=[int]$group[90];firstWidth=[int]$group[6];pngMagic=[Convert]::ToBase64String($firstFrame[0..7]);manifest=$manifest} | ConvertTo-Json -Compress
