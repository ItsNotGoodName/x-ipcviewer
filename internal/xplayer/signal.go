package xplayer

func NewState[T comparable](value T) *State[T] {
	return &State[T]{}
}

type State[T comparable] struct {
	V       T
	effects []func()
}

func (s *State[T]) Update(value T) {
	if s.V == value {
		return
	}
	s.V = value
	for _, fn := range s.effects {
		fn()
	}
}

func (s *State[T]) AddEffect(fn func()) {
	s.effects = append(s.effects, fn)
}
