// Package compute implements the staked-proposer/challenge computation market (Phase 6, SYS-COMP-*),
// grounded in US20240364498A1: a requester posts a task with a bounty; a proposer commits a solution by
// hash and stakes; on challenge both assets are placed under threshold control of a group (released when
// a threshold signs); the challenge is resolved by selecting a solution and assets are distributed. A
// resolved result is fed as an input EVENT into the relevant DFA (SYS-COMP-002).
package compute

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

type phase int

const (
	Posted phase = iota
	Committed
	Challenged
	Resolved
)

// Task is one computation request (e.g. adjudicate a disputed logistics condition).
type Task struct {
	ID         string
	Bounty     uint64 // first digital asset
	phase      phase
	commitHash string // proposer's solution commitment
	stake      uint64 // second digital asset
	solution   string
	// threshold control of the escrowed assets while challenged
	groupSize int
	threshold int
}

type Market struct{ tasks map[string]*Task }

func NewMarket() *Market { return &Market{tasks: map[string]*Task{}} }

func (m *Market) Post(id string, bounty uint64, groupSize, threshold int) (*Task, error) {
	if threshold < 1 || threshold > groupSize {
		return nil, errors.New("bad threshold group")
	}
	t := &Task{ID: id, Bounty: bounty, phase: Posted, groupSize: groupSize, threshold: threshold}
	m.tasks[id] = t
	return t, nil
}

// Commit records a proposer's solution by hash + stake (the solution itself is revealed at resolve).
func (m *Market) Commit(id, solution string, stake uint64) error {
	t, ok := m.tasks[id]
	if !ok || t.phase != Posted {
		return errors.New("commit: bad task/phase")
	}
	h := sha256.Sum256([]byte(solution))
	t.commitHash = hex.EncodeToString(h[:])
	t.stake = stake
	t.solution = solution
	t.phase = Committed
	return nil
}

func (m *Market) Challenge(id string) error {
	t, ok := m.tasks[id]
	if !ok || t.phase != Committed {
		return errors.New("challenge: bad task/phase")
	}
	t.phase = Challenged // assets now under threshold control of the group
	return nil
}

// Result is the resolved outcome and the DFA event to feed (SYS-COMP-002).
type Result struct {
	TaskID    string
	Accepted  bool
	Solution  string
	Payout    uint64 // bounty + stake to the winner
	DFAEvent  string // event injected into the document/consignment DFA
}

// Resolve selects the committed solution iff it matches its commitment and `groupSigs` meets the
// threshold controlling the escrow; distributes assets and returns the DFA event to inject.
func (m *Market) Resolve(id, revealed string, groupSigs int, dfaEventOnAccept string) (Result, error) {
	t, ok := m.tasks[id]
	if !ok || t.phase != Challenged {
		return Result{}, errors.New("resolve: bad task/phase")
	}
	if groupSigs < t.threshold {
		return Result{}, errors.New("resolve: threshold not met (assets stay escrowed)")
	}
	h := sha256.Sum256([]byte(revealed))
	accepted := hex.EncodeToString(h[:]) == t.commitHash
	t.phase = Resolved
	r := Result{TaskID: id, Accepted: accepted, Solution: revealed}
	if accepted {
		r.Payout = t.Bounty + t.stake
		r.DFAEvent = dfaEventOnAccept
	}
	return r, nil
}
