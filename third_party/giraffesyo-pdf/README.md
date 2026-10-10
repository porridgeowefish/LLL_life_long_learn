# Local compatibility patch of giraffesyo/pdf v0.7.0

Upstream: https://github.com/giraffesyo/pdf/tree/v0.7.0 (MIT; see LICENSE).
Runtime source and pdftest helpers are copied unchanged except for:

- font.go and cid_unicode.go: after explicit ToUnicode mappings, resolve Adobe
  GB1/CNS1/Japan1/Korea1 CIDs using compiled official CID-to-Unicode maps. Unknown
  collections retain upstream font fallback. No network or host fonts are used.
- extract.go: image-painting limits apply when collecting images/OCR. Text-only
  extraction still counts paintings and enforces operator, stream, glyph,
  recursion, input, output and cancellation limits without discarding text on
  pages that paint many images.

The four CMaps in cmaps/ are unmodified resources from
https://github.com/adobe-type-tools/mapping-resources-pdf/tree/master/pdf2unicode.
They retain their full redistribution notices; the packaged
docs/release/THIRD_PARTY_NOTICES.txt repeats those notices for binary distribution.
Application regressions live in sources/internal/parser/parser_test.go. No learner
documents are included here. Keep patches confined to these compatibility points
when updating upstream; remove the local replacement once upstream supports them.
