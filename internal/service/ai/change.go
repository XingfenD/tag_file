package ai

// SubscribeChange registers one listener for "task or suggestion state changed".
// The channel is buffered with room for exactly one signal and receives a
// non-blocking send per mutation, so a burst of changes coalesces into one
// wake-up: the subscriber's job is to re-read the current state, not to count
// events. The returned cancel removes the subscription and must be called when
// the listener goes away (an SSE connection closing); calling it twice is safe.
func (s *Service) SubscribeChange() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	s.changeMu.Lock()
	if s.changeSubs == nil {
		s.changeSubs = make(map[chan struct{}]struct{})
	}
	s.changeSubs[ch] = struct{}{}
	s.changeMu.Unlock()
	return ch, func() {
		s.changeMu.Lock()
		delete(s.changeSubs, ch)
		s.changeMu.Unlock()
	}
}

// notifyChange pokes every subscriber. Like notifyWake, the send never blocks:
// a full buffer already means "there is a change you have not looked at", so no
// mutation is ever delayed by a slow or stalled listener.
func (s *Service) notifyChange() {
	s.changeMu.Lock()
	defer s.changeMu.Unlock()
	for ch := range s.changeSubs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}