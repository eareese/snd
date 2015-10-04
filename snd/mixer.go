package snd

import "sync"

const mixerbuf = 4

// TODO should mixer be stereo out?
type Mixer struct {
	sync.Mutex
	ins []Sound
	out []float64

	// TODO tmp for waveform
	tmp   [mixerbuf][]float64
	sampl []float64
}

func NewMixer(ins ...Sound) *Mixer {
	m := &Mixer{
		ins:   ins,
		out:   make([]float64, DefaultSampleSize),
		sampl: make([]float64, DefaultSampleSize*mixerbuf),
	}
	for i := range m.tmp {
		m.tmp[i] = make([]float64, DefaultSampleSize)
	}
	return m
}

func (m *Mixer) Append(s Sound) {
	m.ins = append(m.ins, s)
}

func (m *Mixer) Output() []float64 {
	return m.out
}

func (m *Mixer) Prepare() {
	m.Lock()
	defer m.Unlock()

	for _, in := range m.ins {
		in.Prepare()
	}

	for i := range m.out {
		m.out[i] = 0
		for _, in := range m.ins {
			m.out[i] += in.Output()[i]
		}
		m.out[i] /= float64(len(m.ins))
	}

	// TODO for waveform
	buf := m.tmp[0]
	for i := 0; i+1 < len(m.tmp); i++ {
		m.tmp[i] = m.tmp[i+1]
	}
	for i, x := range m.out {
		buf[i] = x
	}
	m.tmp[len(m.tmp)-1] = buf
}

func (m *Mixer) Samples() []float64 {
	m.Lock()
	defer m.Unlock()

	for i, buf := range m.tmp {
		idx := i * DefaultSampleSize
		sl := m.sampl[idx : idx+DefaultSampleSize]
		for j, x := range buf {
			sl[j] = x
		}
	}
	return m.sampl
}
