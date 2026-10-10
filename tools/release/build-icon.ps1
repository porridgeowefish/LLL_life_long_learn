param([string]$OutputDirectory = '')
$ErrorActionPreference = 'Stop'
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) { $OutputDirectory = Join-Path $PSScriptRoot '..\..\scripts\assets' }
$OutputDirectory = [IO.Path]::GetFullPath($OutputDirectory)
[IO.Directory]::CreateDirectory($OutputDirectory) | Out-Null
Add-Type -AssemblyName System.Drawing
Add-Type -ReferencedAssemblies System.Drawing -TypeDefinition @'
using System;
using System.Drawing;
using System.Drawing.Drawing2D;
using System.Drawing.Imaging;
using System.IO;
public static class LearnAppIcon {
  static GraphicsPath Book(bool right) {
    var p=new GraphicsPath();
    if(!right) { p.StartFigure();p.AddBezier(57,85,80,77,104,81,128,94);p.AddLine(128,94,128,179);p.AddBezier(128,179,103,165,80,162,57,171);p.CloseFigure(); }
    else {p.StartFigure();p.AddBezier(128,94,152,81,176,77,199,85);p.AddLine(199,85,199,171);p.AddBezier(199,171,176,162,153,165,128,179);p.CloseFigure();}
    return p;
  }
  static byte[] Render(int size) {
    using(var image=new Bitmap(size,size,PixelFormat.Format32bppArgb))
    using(var g=Graphics.FromImage(image)) {
      g.Clear(Color.Transparent);g.SmoothingMode=SmoothingMode.AntiAlias;
      g.PixelOffsetMode=PixelOffsetMode.HighQuality;g.ScaleTransform(size/256f,size/256f);
      using(var tile=new GraphicsPath()) {
        tile.AddArc(12,12,108,108,180,90);tile.AddArc(136,12,108,108,270,90);
        tile.AddArc(136,136,108,108,0,90);tile.AddArc(12,136,108,108,90,90);tile.CloseFigure();
        using(var fill=new LinearGradientBrush(new Point(12,12),new Point(244,244),ColorTranslator.FromHtml("#60775F"),ColorTranslator.FromHtml("#344D42")))g.FillPath(fill,tile);
      }
      using(var left=Book(false))using(var fill=new SolidBrush(ColorTranslator.FromHtml("#F5F0DF")))g.FillPath(fill,left);
      using(var right=Book(true))using(var fill=new SolidBrush(ColorTranslator.FromHtml("#E5ECD8")))g.FillPath(fill,right);
      using(var spine=new Pen(ColorTranslator.FromHtml("#81967B"),5))g.DrawLine(spine,128,97,128,178);
      using(var pen=new Pen(ColorTranslator.FromHtml("#647D64"),5)) {
        pen.StartCap=LineCap.Round;pen.EndCap=LineCap.Round;
        for(int y=105;y<=139;y+=17){g.DrawBezier(pen,74,y,88,y-2,100,y+1,111,y+6);g.DrawBezier(pen,145,y+6,157,y+1,169,y-2,182,y);}
      }
      using(var fill=new SolidBrush(ColorTranslator.FromHtml("#EBC981")))g.FillPolygon(fill,new PointF[]{new PointF(181,44),new PointF(185,55),new PointF(196,59),new PointF(185,63),new PointF(181,74),new PointF(177,63),new PointF(166,59),new PointF(177,55)});
      using(var memory=new MemoryStream()){image.Save(memory,ImageFormat.Png);return memory.ToArray();}
    }
  }
  public static void Build(string directory) {
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
      using(var ink=new SolidBrush(ColorTranslator.FromHtml("#344D42")))
      using(var muted=new SolidBrush(ColorTranslator.FromHtml("#6C746B"))) {
        g.Clear(ColorTranslator.FromHtml("#FCFBF7"));g.InterpolationMode=InterpolationMode.HighQualityBicubic;
        g.DrawImage(icon,18,8,42,42);g.DrawString("Learn-With-AI",heading,ink,70,9);
        g.DrawString("Your learning workspace",caption,muted,71,32);
        banner.Save(Path.Combine(directory,"installer-banner.bmp"),ImageFormat.Bmp);
      }
      using(var dialog=new Bitmap(493,312,PixelFormat.Format24bppRgb))
      using(var g=Graphics.FromImage(dialog))
      using(var background=new LinearGradientBrush(new Point(0,0),new Point(164,312),ColorTranslator.FromHtml("#60775F"),ColorTranslator.FromHtml("#344D42")))
      using(var heading=new Font("Segoe UI",16,FontStyle.Bold,GraphicsUnit.Pixel))
      using(var caption=new Font("Segoe UI",12,FontStyle.Regular,GraphicsUnit.Pixel))
      using(var white=new SolidBrush(ColorTranslator.FromHtml("#F5F0DF")))
      using(var line=new Pen(ColorTranslator.FromHtml("#9BA88F"),1)) {
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
[LearnAppIcon]::Build($OutputDirectory)
Write-Output "Created multi-resolution application icon: $OutputDirectory\learn-with-ai.ico"
