param([string]$OutputDirectory = '', [string]$SourceImage = '')
$ErrorActionPreference = 'Stop'
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) { $OutputDirectory = Join-Path $PSScriptRoot '..\..\scripts\assets' }
$OutputDirectory = [IO.Path]::GetFullPath($OutputDirectory)
if ([string]::IsNullOrWhiteSpace($SourceImage)) { $SourceImage = Join-Path $PSScriptRoot '..\..\frontend\public\app-icon.png' }
$SourceImage = [IO.Path]::GetFullPath($SourceImage)
if (-not (Test-Path -LiteralPath $SourceImage -PathType Leaf)) { throw "Missing selected logo: $SourceImage" }
[IO.Directory]::CreateDirectory($OutputDirectory) | Out-Null
Add-Type -AssemblyName System.Drawing
Add-Type -ReferencedAssemblies System.Drawing -TypeDefinition @'
using System;
using System.Drawing;
using System.Drawing.Drawing2D;
using System.Drawing.Imaging;
using System.IO;
public static class LearnAppIcon {
  static string sourceImage;
  static byte[] Render(int size) {
    using(var source=new Bitmap(sourceImage))
    using(var image=new Bitmap(size,size,PixelFormat.Format32bppArgb))
    using(var g=Graphics.FromImage(image)) {
      g.Clear(Color.Transparent);
      g.InterpolationMode=InterpolationMode.HighQualityBicubic;
      g.PixelOffsetMode=PixelOffsetMode.HighQuality;
      g.CompositingQuality=CompositingQuality.HighQuality;
      g.DrawImage(source,new Rectangle(0,0,size,size),0,0,source.Width,source.Height,GraphicsUnit.Pixel);
      using(var memory=new MemoryStream()){image.Save(memory,ImageFormat.Png);return memory.ToArray();}
    }
  }
  public static void Build(string directory, string source) {
    sourceImage=source;
    int[] sizes={16,24,32,48,64,128,256};
    byte[][] frames=new byte[sizes.Length][];
    for(int i=0;i<sizes.Length;i++)frames[i]=Render(sizes[i]);
    using(var file=File.Create(Path.Combine(directory,"learn-with-ai.ico")))
    using(var writer=new BinaryWriter(file)){
      writer.Write((ushort)0);writer.Write((ushort)1);writer.Write((ushort)sizes.Length);
      uint offset=(uint)(6+16*sizes.Length);
      for(int i=0;i<sizes.Length;i++){writer.Write((byte)(sizes[i]==256?0:sizes[i]));writer.Write((byte)(sizes[i]==256?0:sizes[i]));writer.Write((byte)0);writer.Write((byte)0);writer.Write((ushort)1);writer.Write((ushort)32);writer.Write((uint)frames[i].Length);writer.Write(offset);offset+=(uint)frames[i].Length;}
      foreach(var frame in frames)writer.Write(frame);
    }
    File.WriteAllBytes(Path.Combine(directory,"learn-with-ai.png"),Render(256));
    using(var icon=new Bitmap(new MemoryStream(Render(256)))) {
      using(var banner=new Bitmap(493,58,PixelFormat.Format24bppRgb))
      using(var g=Graphics.FromImage(banner))
      using(var heading=new Font("Segoe UI",16,FontStyle.Bold,GraphicsUnit.Pixel))
      using(var caption=new Font("Segoe UI",10,FontStyle.Regular,GraphicsUnit.Pixel))
      using(var ink=new SolidBrush(ColorTranslator.FromHtml("#314A63")))
      using(var muted=new SolidBrush(ColorTranslator.FromHtml("#64727E"))) {
        g.Clear(ColorTranslator.FromHtml("#FCFBF7"));g.InterpolationMode=InterpolationMode.HighQualityBicubic;
        g.DrawImage(icon,18,8,42,42);g.DrawString("Learn-With-AI",heading,ink,70,9);
        g.DrawString("Your learning workspace",caption,muted,71,32);
        banner.Save(Path.Combine(directory,"installer-banner.bmp"),ImageFormat.Bmp);
      }
      using(var dialog=new Bitmap(493,312,PixelFormat.Format24bppRgb))
      using(var g=Graphics.FromImage(dialog))
      using(var background=new LinearGradientBrush(new Point(0,0),new Point(164,312),ColorTranslator.FromHtml("#8CAAC1"),ColorTranslator.FromHtml("#314A63")))
      using(var heading=new Font("Segoe UI",16,FontStyle.Bold,GraphicsUnit.Pixel))
      using(var caption=new Font("Segoe UI",12,FontStyle.Regular,GraphicsUnit.Pixel))
      using(var white=new SolidBrush(ColorTranslator.FromHtml("#F5F0DF")))
      using(var line=new Pen(ColorTranslator.FromHtml("#ACBED0"),1)) {
        g.Clear(ColorTranslator.FromHtml("#FCFBF7"));g.FillRectangle(background,0,0,164,312);
        g.InterpolationMode=InterpolationMode.HighQualityBicubic;g.DrawImage(icon,30,68,104,104);
        g.DrawString("Learn-With-AI",heading,white,27,191);g.DrawLine(line,28,222,135,222);
        g.DrawString("Learn. Ask. Think.",caption,white,28,237);
        dialog.Save(Path.Combine(directory,"installer-dialog.bmp"),ImageFormat.Bmp);
      }
    }
  }
}
'@
[LearnAppIcon]::Build($OutputDirectory, $SourceImage)
Write-Output "Created multi-resolution application icon: $OutputDirectory\learn-with-ai.ico"
