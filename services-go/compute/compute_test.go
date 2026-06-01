package compute

import "testing"

func TestStakedComputationResolvesAndFeedsDFA(t *testing.T) {
	m := NewMarket()
	// requester posts a dispute-adjudication task; escrow under 2-of-3 group threshold control
	_, err := m.Post("dispute#1", 1000, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Commit("dispute#1", "condition-OK", 500); err != nil {
		t.Fatal(err)
	}
	if err := m.Challenge("dispute#1"); err != nil {
		t.Fatal(err)
	}

	// below threshold: assets stay escrowed, no resolution
	if _, err := m.Resolve("dispute#1", "condition-OK", 1, "resolve"); err == nil {
		t.Fatal("must not resolve below threshold")
	}
	// threshold met + correct reveal -> accepted, payout = bounty+stake, DFA event emitted
	r, err := m.Resolve("dispute#1", "condition-OK", 2, "resolve")
	if err != nil {
		t.Fatal(err)
	}
	if !r.Accepted || r.Payout != 1500 || r.DFAEvent != "resolve" {
		t.Fatalf("unexpected result: %+v", r)
	}
}

func TestResolveRejectsBadReveal(t *testing.T) {
	m := NewMarket()
	if _, err := m.Post("d2", 100, 1, 1); err != nil {
		t.Fatal(err)
	}
	if err := m.Commit("d2", "secret-soln", 50); err != nil {
		t.Fatal(err)
	}
	if err := m.Challenge("d2"); err != nil {
		t.Fatal(err)
	}
	r, err := m.Resolve("d2", "WRONG", 1, "resolve")
	if err != nil {
		t.Fatal(err)
	}
	if r.Accepted || r.DFAEvent != "" {
		t.Fatalf("mismatched reveal must not be accepted: %+v", r)
	}
}
