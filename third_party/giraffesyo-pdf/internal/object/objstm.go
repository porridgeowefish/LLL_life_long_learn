package object

import (
	"errors"
	"fmt"
)

// objStm is a decoded object stream (ISO 32000-1 §7.5.7): a header of
// (object-number, offset) pairs followed by the packed object bodies.
type objStm struct {
	nums    []int
	offsets []int
	body    []byte
	first   int
}

// objStmCache holds a few decoded object streams so that accessing many
// objects from the same container does not re-inflate it each time.
type objStmCache struct {
	m     map[int]*objStm
	order []int
}

func newObjStmCache() *objStmCache {
	return &objStmCache{m: map[int]*objStm{}}
}

func (c *objStmCache) get(num int) (*objStm, bool) {
	s, ok := c.m[num]
	return s, ok
}

func (c *objStmCache) put(num int, s *objStm) {
	if _, ok := c.m[num]; !ok {
		c.order = append(c.order, num)
		for len(c.order) > maxObjStmCache {
			evict := c.order[0]
			c.order = c.order[1:]
			delete(c.m, evict)
		}
	}
	c.m[num] = s
}

// objStmObject returns object wantNum, the idx-th entry packed in object
// stream stmNum.
func (r *Reader) objStmObject(stmNum, idx, wantNum int) (any, error) {
	os, err := r.loadObjStm(stmNum)
	if err != nil {
		return nil, err
	}
	if idx < 0 || idx >= len(os.nums) {
		return nil, fmt.Errorf("pdf: object stream index %d out of range", idx)
	}
	if os.nums[idx] != wantNum {
		return nil, fmt.Errorf("pdf: object stream entry %d is object %d, not %d", idx, os.nums[idx], wantNum)
	}
	start := os.first + os.offsets[idx]
	end := len(os.body)
	if idx+1 < len(os.offsets) {
		end = os.first + os.offsets[idx+1]
	}
	if start < 0 || end > len(os.body) || start > end {
		return nil, errors.New("pdf: object stream entry out of bounds")
	}
	// Packed objects carry no "N G obj" wrapper and are already decrypted
	// (the container stream was decrypted as a whole), so parse the body
	// slice directly with no string-decryption pass.
	p := &objParser{lex: newLexer(os.body[start:end]), r: r}
	v, perr := p.parseObject(0)
	if perr != nil {
		return nil, perr
	}
	return v, nil
}

// loadObjStm decodes and caches an object stream by number.
func (r *Reader) loadObjStm(stmNum int) (*objStm, error) {
	if os, ok := r.objStms.get(stmNum); ok {
		return os, nil
	}
	if stmNum < 0 || stmNum >= len(r.xref) {
		return nil, fmt.Errorf("pdf: object stream %d out of range", stmNum)
	}
	if r.xref[stmNum].kind != 1 {
		return nil, errors.New("pdf: object stream is not a direct object")
	}
	obj, err := r.parseObjectAt(r.xref[stmNum].offset, stmNum, 0)
	if err != nil {
		return nil, err
	}
	s, ok := obj.(*stream)
	if !ok {
		return nil, errors.New("pdf: /ObjStm reference is not a stream")
	}
	data, err := r.rawStreamBytes(s)
	if err != nil {
		return nil, err
	}

	n := int(r.asInt(s.d["N"]))
	first := int(r.asInt(s.d["First"]))
	if n < 0 || n > maxObjStmObjs || first < 0 || first > len(data) {
		return nil, errors.New("pdf: object stream /N or /First out of range")
	}
	os := &objStm{body: data, first: first, nums: make([]int, 0, n), offsets: make([]int, 0, n)}
	lex := newLexer(data[:first])
	for range n {
		numTok := lex.next()
		offTok := lex.next()
		if numTok.kind != tokInt || offTok.kind != tokInt {
			return nil, errors.New("pdf: malformed object stream header")
		}
		os.nums = append(os.nums, int(numTok.num))
		os.offsets = append(os.offsets, int(offTok.num))
	}
	r.objStms.put(stmNum, os)
	return os, nil
}
