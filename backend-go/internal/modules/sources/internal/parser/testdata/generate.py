"""Regenerate genuine fixtures with python-docx/python-pptx/openpyxl/reportlab/fonttools.

Development-only tools; the shipped parser has no Python dependency. All text
and the tiny valid TrueType font below are original fixtures (CC0).
"""
from pathlib import Path
from tempfile import TemporaryDirectory

from docx import Document
from pptx import Presentation
from openpyxl import Workbook
from reportlab.pdfgen.canvas import Canvas
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont
from fontTools.fontBuilder import FontBuilder
from fontTools.pens.ttGlyphPen import TTGlyphPen

root = Path(__file__).parent
doc = Document()
doc.add_heading('中文学习资料', 1)
doc.add_paragraph('保留第一段。')
table = doc.add_table(rows=1, cols=2)
table.rows[0].cells[0].text = '概念'
table.rows[0].cells[1].text = '定义'
doc.sections[0].header.paragraphs[0].text = '页眉资料'
doc.save(root / 'lesson.docx')

slides = Presentation()
for title in ('第二张先讲', '第一张后讲'):
    slide = slides.slides.add_slide(slides.slide_layouts[1])
    slide.shapes.title.text = title
    slide.placeholders[1].text = '原生幻灯片正文'
    slide.notes_slide.notes_text_frame.text = '讲者备注'
slides.save(root / 'lesson.pptx')

book = Workbook()
sheet = book.active
sheet.title = '中文预算'
sheet.append(['项目', '金额'])
sheet.append(['资料', 120])
sheet['C3'] = '=SUM(B2:B2)'
book.create_sheet('第二页')['A1'] = '第二表数据'
book.save(root / 'lesson.xlsx')

# Build a valid 5-glyph TTF with a Unicode cmap. Original geometric test
# glyphs deliberately avoid any dependency on a proprietary installed font.
with TemporaryDirectory() as temporary:
    font_path = Path(temporary) / 'Fixture.ttf'
    fb = FontBuilder(1000, isTTF=True)
    names = ['.notdef', 'space', 'uni4E2D', 'uni6587', 'uni5B66', 'uni4E60']
    fb.setupGlyphOrder(names)
    fb.setupCharacterMap({32: 'space', 0x4e2d: 'uni4E2D', 0x6587: 'uni6587', 0x5b66: 'uni5B66', 0x4e60: 'uni4E60'})
    glyphs = {}
    for index, name in enumerate(names):
        pen = TTGlyphPen(None)
        if name != 'space':
            pen.moveTo((100, 100)); pen.lineTo((100, 800)); pen.lineTo((700 + index * 10, 800)); pen.lineTo((700 + index * 10, 100)); pen.closePath()
        glyphs[name] = pen.glyph()
    fb.setupGlyf(glyphs)
    fb.setupHorizontalMetrics({name: (1000, 0) for name in names})
    fb.setupHorizontalHeader(ascent=900, descent=-100)
    fb.setupNameTable({'familyName': 'LLL Fixture', 'styleName': 'Regular', 'uniqueFontIdentifier': 'LLLFixture-CC0', 'fullName': 'LLL Fixture Regular', 'psName': 'LLLFixture-Regular'})
    fb.setupOS2(sTypoAscender=900, sTypoDescender=-100, usWinAscent=900, usWinDescent=100)
    fb.setupPost(); fb.setupMaxp(); fb.save(font_path)
    pdfmetrics.registerFont(TTFont('LLLFixture', str(font_path)))
    canvas = Canvas(str(root / 'embedded-cjk.pdf'), pageCompression=1, invariant=1)
    canvas.setFont('LLLFixture', 18)
    canvas.drawString(50, 700, '中文学习')
    canvas.beginForm('nested', 0, 0, 500, 500)
    canvas.setFont('LLLFixture', 18)
    canvas.drawString(50, 300, '学习中文')
    canvas.endForm()
    canvas.doForm('nested')
    canvas.showPage()
    canvas.setFont('Helvetica', 14)
    canvas.drawString(50, 700, 'Second page through the callback')
    canvas.save()

canvas = Canvas(str(root / 'textless.pdf'), invariant=1)
canvas.rect(50, 50, 150, 150)
canvas.save()
