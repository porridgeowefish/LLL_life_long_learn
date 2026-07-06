package runprogress

import "testing"

func TestSetValidatesToken(t *testing.T) {
	s := New()
	tok := s.Register("r1")
	st, ok := s.Set("r1", "wrong", Status{Activity: "x"})
	if ok {
		t.Errorf("wrong token must be rejected")
	}
	st, ok = s.Set("r1", tok, Status{Activity: "wrote p", PagesDone: 2, PagesPlanned: 5})
	if !ok || st.Activity != "wrote p" || st.PagesDone != 2 {
		t.Errorf("valid set failed: %+v ok=%v", st, ok)
	}
	got, has := s.Get("r1")
	if !has || got.PagesPlanned != 5 {
		t.Errorf("get failed: %+v", got)
	}
	// unknown runId rejects even with empty token
	if _, ok := s.Set("nope", "", Status{}); ok {
		t.Errorf("unknown runId must reject")
	}
}

func TestRegisterReturnsTokenAndAllowsReRegister(t *testing.T) {
	s := New()
	tok := s.Register("r1")
	if tok == "" {
		t.Fatal("token must not be empty")
	}
	// Registering again issues a new token; the old one must stop working.
	tok2 := s.Register("r1")
	if tok2 == "" || tok2 == tok {
		t.Fatalf("re-register must issue a fresh token: first=%q second=%q", tok, tok2)
	}
	if _, ok := s.Set("r1", tok, Status{Activity: "old"}); ok {
		t.Errorf("stale token must be rejected after re-register")
	}
	st, ok := s.Set("r1", tok2, Status{Activity: "new", PagesDone: 1})
	if !ok || st.Activity != "new" {
		t.Errorf("fresh token must work: %+v ok=%v", st, ok)
	}
}

func TestSetMergesNonZeroFields(t *testing.T) {
	s := New()
	tok := s.Register("r1")
	// First report sets pagesPlanned + activity.
	first, ok := s.Set("r1", tok, Status{Activity: "planning", PagesPlanned: 5})
	if !ok || first.PagesPlanned != 5 || first.Activity != "planning" || first.PagesDone != 0 {
		t.Errorf("first set wrong: %+v ok=%v", first, ok)
	}
	// Second report sends only pagesDone — pagesPlanned must persist.
	second, ok := s.Set("r1", tok, Status{PagesDone: 2})
	if !ok || second.PagesDone != 2 || second.PagesPlanned != 5 || second.Activity != "planning" {
		t.Errorf("merge did not preserve prior fields: %+v ok=%v", second, ok)
	}
	// Done flips the flag and is sticky.
	third, ok := s.Set("r1", tok, Status{Done: true})
	if !ok || !third.Done || third.PagesPlanned != 5 {
		t.Errorf("done merge wrong: %+v ok=%v", third, ok)
	}
}

func TestGetUnknownReturnsFalse(t *testing.T) {
	s := New()
	if _, ok := s.Get("missing"); ok {
		t.Errorf("Get on unknown runId must return false")
	}
}
