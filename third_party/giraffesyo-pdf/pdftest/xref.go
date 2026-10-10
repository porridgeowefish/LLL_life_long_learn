package pdftest

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"strings"
)

// Flate renders a stream object whose body is zlib-compressed, emitting
// /Filter /FlateDecode and the correct /Length.
func Flate(dict, content string) string {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	_, _ = w.Write([]byte(content))
	_ = w.Close()
	comp := b.String()
	extra := strings.TrimSpace(dict)
	if extra != "" {
		extra += " "
	}
	return fmt.Sprintf("<< %s/Filter /FlateDecode /Length %d >>\nstream\n%s\nendstream", extra, len(comp), comp)
}

// StreamWithLength renders a stream object with an explicit (possibly
// wrong) /Length, for exercising length recovery.
func StreamWithLength(dict, content string, length int) string {
	extra := strings.TrimSpace(dict)
	if extra != "" {
		extra += " "
	}
	return fmt.Sprintf("<< %s/Length %d >>\nstream\n%s\nendstream", extra, length, content)
}

// BuildXrefStream assembles a PDF whose cross-reference data is a PDF 1.5
// xref stream (no classic table). objs[i] becomes object i+1; the xref
// stream is appended as the final object. rootID is the catalog.
func BuildXrefStream(rootID int, objs ...string) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.5\n")
	offsets := make([]int, len(objs)+2) // +1 for the xref stream object
	for i, obj := range objs {
		offsets[i+1] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xrefNum := len(objs) + 1
	xrefOff := b.Len()
	offsets[xrefNum] = xrefOff

	// Rows for objects 0..xrefNum: type/field2/field3 with /W [1 4 2].
	var rows bytes.Buffer
	writeRow(&rows, 0, 0, 65535) // object 0: free head
	for i := 1; i <= len(objs); i++ {
		writeRow(&rows, 1, offsets[i], 0)
	}
	writeRow(&rows, 1, xrefOff, 0) // the xref stream itself

	fmt.Fprintf(&b, "%d 0 obj\n<< /Type /XRef /Size %d /Root %d 0 R /W [1 4 2] /Length %d >>\nstream\n",
		xrefNum, xrefNum+1, rootID, rows.Len())
	b.Write(rows.Bytes())
	b.WriteString("\nendstream\nendobj\n")
	fmt.Fprintf(&b, "startxref\n%d\n%%%%EOF\n", xrefOff)
	return b.Bytes()
}

// BuildObjStm assembles a PDF that packs the objects named in `packed`
// (1-based object numbers) into one object stream, with an xref stream
// referencing them by type-2 entries. Objects not listed in `packed`
// remain direct. Stream objects cannot be packed.
func BuildObjStm(rootID int, packed []int, objs ...string) []byte {
	isPacked := map[int]bool{}
	for _, n := range packed {
		isPacked[n] = true
	}

	var b bytes.Buffer
	b.WriteString("%PDF-1.5\n")
	nObjs := len(objs)
	objStmNum := nObjs + 1
	xrefNum := nObjs + 2
	offsets := make([]int, xrefNum+1)

	// Build the object-stream body: header pairs then packed bodies.
	var bodies bytes.Buffer
	var header strings.Builder
	idxInStm := map[int]int{}
	stmIdx := 0
	for i, obj := range objs {
		num := i + 1
		if !isPacked[num] {
			continue
		}
		fmt.Fprintf(&header, "%d %d ", num, bodies.Len())
		bodies.WriteString(obj)
		bodies.WriteByte('\n')
		idxInStm[num] = stmIdx
		stmIdx++
	}
	first := header.Len()
	objStmBody := header.String() + bodies.String()

	// Emit direct objects.
	for i, obj := range objs {
		num := i + 1
		if isPacked[num] {
			continue
		}
		offsets[num] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", num, obj)
	}
	// Emit the object stream.
	offsets[objStmNum] = b.Len()
	fmt.Fprintf(&b, "%d 0 obj\n<< /Type /ObjStm /N %d /First %d /Length %d >>\nstream\n%s\nendstream\nendobj\n",
		objStmNum, stmIdx, first, len(objStmBody), objStmBody)

	// Emit the xref stream.
	xrefOff := b.Len()
	offsets[xrefNum] = xrefOff
	var rows bytes.Buffer
	writeRow(&rows, 0, 0, 65535)
	for num := 1; num <= nObjs; num++ {
		if isPacked[num] {
			writeRow(&rows, 2, objStmNum, idxInStm[num])
		} else {
			writeRow(&rows, 1, offsets[num], 0)
		}
	}
	writeRow(&rows, 1, offsets[objStmNum], 0)
	writeRow(&rows, 1, xrefOff, 0)

	fmt.Fprintf(&b, "%d 0 obj\n<< /Type /XRef /Size %d /Root %d 0 R /W [1 4 2] /Length %d >>\nstream\n",
		xrefNum, xrefNum+1, rootID, rows.Len())
	b.Write(rows.Bytes())
	b.WriteString("\nendstream\nendobj\n")
	fmt.Fprintf(&b, "startxref\n%d\n%%%%EOF\n", xrefOff)
	return b.Bytes()
}

// BuildHybrid assembles a hybrid-reference file: a classic xref table
// plus a /XRefStm pointing at a supplementary xref stream that alone
// locates the last object. A reader ignoring /XRefStm loses that object.
func BuildHybrid(rootID int, objs ...string) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.5\n")
	offsets := make([]int, len(objs)+1)
	for i, obj := range objs {
		offsets[i+1] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	// The last object is recorded only in the xref stream.
	hidden := len(objs)

	// Supplementary xref stream covering the hidden object.
	xrefStmNum := len(objs) + 1
	xrefStmOff := b.Len()
	var rows bytes.Buffer
	writeRow(&rows, 1, offsets[hidden], 0)
	fmt.Fprintf(&b, "%d 0 obj\n<< /Type /XRef /Size %d /Index [%d 1] /W [1 4 2] /Root %d 0 R /Length %d >>\nstream\n",
		xrefStmNum, xrefStmNum+1, hidden, rootID, rows.Len())
	b.Write(rows.Bytes())
	b.WriteString("\nendstream\nendobj\n")

	// Classic table covering objects 0..hidden-1 (not the hidden one).
	classicOff := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", hidden)
	for i := 1; i < hidden; i++ {
		fmt.Fprintf(&b, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root %d 0 R /XRefStm %d >>\nstartxref\n%d\n%%%%EOF\n",
		xrefStmNum+1, rootID, xrefStmOff, classicOff)
	return b.Bytes()
}

// writeRow writes one xref-stream row with widths [1 4 2]. Fields are
// ints for caller convenience; test fixtures stay well within range.
func writeRow(b *bytes.Buffer, t byte, f2, f3 int) {
	b.WriteByte(t)
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], uint32(f2&0xFFFFFFFF))
	b.Write(buf[:])
	b.WriteByte(byte(f3 >> 8 & 0xFF))
	b.WriteByte(byte(f3 & 0xFF))
}
