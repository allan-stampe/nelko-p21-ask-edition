param([Parameter(Mandatory=$true)][string]$Exe)
$ErrorActionPreference = 'Stop'
$path = (Resolve-Path $Exe).Path
$version = [System.Diagnostics.FileVersionInfo]::GetVersionInfo($path)
if ($version.FileVersion -ne '1.0.2.0' -or $version.ProductVersion -ne '1.0.2.0') {
    throw "Incorrect Windows version metadata: $($version.FileVersion) / $($version.ProductVersion)"
}
Add-Type @'
using System;
using System.Runtime.InteropServices;
public static class AskIconCheck {
    [StructLayout(LayoutKind.Sequential)] public struct IconInfo {
        [MarshalAs(UnmanagedType.Bool)] public bool IsIcon;
        public uint X; public uint Y; public IntPtr Mask; public IntPtr Color;
    }
    [StructLayout(LayoutKind.Sequential)] public struct Bitmap {
        public int Type; public int Width; public int Height; public int WidthBytes;
        public ushort Planes; public ushort BitsPixel; public IntPtr Bits;
    }
    [DllImport("user32.dll", CharSet=CharSet.Unicode, EntryPoint="PrivateExtractIconsW", SetLastError=true)]
    public static extern uint Extract(string file, int index, int cx, int cy, IntPtr[] icons, uint[] ids, uint count, uint flags);
    [DllImport("user32.dll")] public static extern bool GetIconInfo(IntPtr icon, out IconInfo info);
    [DllImport("gdi32.dll", EntryPoint="GetObjectW", ExactSpelling=true)] public static extern int GetObjectW(IntPtr handle, int size, out Bitmap bitmap);
    [DllImport("gdi32.dll")] public static extern bool DeleteObject(IntPtr handle);
    [DllImport("user32.dll")] public static extern bool DestroyIcon(IntPtr icon);
}
'@
foreach ($size in @(16,32,48,256)) {
    $icons = New-Object IntPtr[] 1
    $ids = New-Object uint32[] 1
    $count = [AskIconCheck]::Extract($path,0,$size,$size,$icons,$ids,1,0)
    if ($count -ne 1 -or $icons[0] -eq [IntPtr]::Zero) { throw "Windows could not extract $size icon" }
    $info = New-Object AskIconCheck+IconInfo
    try {
        if (-not [AskIconCheck]::GetIconInfo($icons[0],[ref]$info)) { throw 'GetIconInfo failed' }
        $bitmap = New-Object AskIconCheck+Bitmap
        if ([AskIconCheck]::GetObjectW($info.Color,[Runtime.InteropServices.Marshal]::SizeOf($bitmap),[ref]$bitmap) -eq 0) { throw 'GetObject failed' }
        if ($bitmap.Width -ne $size -or $bitmap.Height -ne $size) { throw "Wrong extracted icon size: $($bitmap.Width)x$($bitmap.Height)" }
        Write-Host "Windows loaded ${size}x${size} icon successfully"
    } finally {
        if ($info.Color -ne [IntPtr]::Zero) { [void][AskIconCheck]::DeleteObject($info.Color) }
        if ($info.Mask -ne [IntPtr]::Zero) { [void][AskIconCheck]::DeleteObject($info.Mask) }
        [void][AskIconCheck]::DestroyIcon($icons[0])
    }
}
Write-Host "Windows version $($version.FileVersion) and native icon extraction verified."
