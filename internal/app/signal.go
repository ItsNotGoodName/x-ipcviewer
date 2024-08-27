package app

func NewSignal[T comparable](value T) *Signal[T] {
	return &Signal[T]{}
}

type Signal[T comparable] struct {
	V       T
	effects []func()
}

func (s *Signal[T]) SetValue(value T) {
	if s.V == value {
		return
	}
	s.V = value
	for _, fn := range s.effects {
		fn()
	}
}

func (s *Signal[T]) AddEffect(fn func()) {
	s.effects = append(s.effects, fn)
}
