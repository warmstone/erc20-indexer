package indexer

import "sync"

const growAfterSuccesses = 3

type BatchSizer struct {
	max    uint64
	min    uint64
	states map[string]*batchState
	mu     sync.Mutex
}

type batchState struct {
	current   uint64
	successes int
}

func NewBatchSizer(maxBatchSize, minBatchSize uint64) *BatchSizer {
	if maxBatchSize == 0 {
		maxBatchSize = 1000
	}
	if minBatchSize == 0 {
		minBatchSize = 1
	}
	if minBatchSize > maxBatchSize {
		minBatchSize = maxBatchSize
	}
	return &BatchSizer{
		max:    maxBatchSize,
		min:    minBatchSize,
		states: make(map[string]*batchState),
	}
}

func (s *BatchSizer) NextRange(key string, lastIndexed, safeBlock uint64) (uint64, uint64, bool) {
	from := lastIndexed + 1
	if from > safeBlock {
		return 0, 0, false
	}
	size := s.current(key)
	to := from + size - 1
	if to > safeBlock {
		to = safeBlock
	}
	return from, to, true
}

func (s *BatchSizer) RecordSuccess(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.state(key)
	if state.current >= s.max {
		state.current = s.max
		state.successes = 0
		return
	}

	state.successes++
	if state.successes < growAfterSuccesses {
		return
	}
	state.current *= 2
	if state.current > s.max {
		state.current = s.max
	}
	state.successes = 0
}

func (s *BatchSizer) RecordFailure(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.state(key)
	state.successes = 0
	next := state.current / 2
	if next < s.min {
		next = s.min
	}
	state.current = next
}

func (s *BatchSizer) current(key string) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state(key).current
}

func (s *BatchSizer) state(key string) *batchState {
	state, ok := s.states[key]
	if ok {
		return state
	}
	state = &batchState{current: s.max}
	s.states[key] = state
	return state
}
