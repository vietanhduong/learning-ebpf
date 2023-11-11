package helpers

type Callback struct {
	RawFn  func(raw []byte, size int32)
	LostFn func(uint64)
}

func NewCallback(raw func([]byte, int32), lost func(uint64)) *Callback {
	return &Callback{
		RawFn:  raw,
		LostFn: lost,
	}
}

func (cb *Callback) RawSample(raw []byte, size int32) {
	if cb.RawFn != nil {
		cb.RawFn(raw, size)
	}
}

func (cb *Callback) LostSamples(count uint64) {
	if cb.LostFn != nil {
		cb.LostFn(count)
	}
}
