package snd

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/exp/f32"
	"golang.org/x/mobile/exp/gl/glutil"
	"golang.org/x/mobile/gl"
)

// TODO this is intended to graphically represent sound using opengl
// but the package is "snd". It doesn't make much sense to require
// go mobile gl to build snd (increasing complexity of portability)
// so move this to a subpkg requiring explicit importing.

const epsilon = 0.0001

func equals(a, b float64) bool {
	return (a-b) < epsilon && (b-a) < epsilon
}

type Waveform struct {
	program  gl.Program
	position gl.Attrib
	color    gl.Uniform
	buf      gl.Buffer

	in *Mixer

	align    bool
	alignamp float64
	aligned  []float64
}

// TODO just how many samples do we want/need to display something useful?
func NewWaveform(in *Mixer, ctx gl.Context) (*Waveform, error) {
	wf := &Waveform{
		in:      in,
		aligned: make([]float64, DefaultSampleSize*mixerbuf),
	}

	var err error
	wf.program, err = glutil.CreateProgram(ctx, vertexShader, fragmentShader)
	if err != nil {
		return nil, fmt.Errorf("error creating GL program: %v", err)
	}

	// create and alloc hw buf
	wf.buf = ctx.CreateBuffer()
	ctx.BindBuffer(gl.ARRAY_BUFFER, wf.buf)
	ctx.BufferData(gl.ARRAY_BUFFER, make([]byte, len(wf.aligned)*12), gl.STREAM_DRAW)

	wf.position = ctx.GetAttribLocation(wf.program, "position")
	wf.color = ctx.GetUniformLocation(wf.program, "color")
	return wf, nil
}

func (wf *Waveform) Align(amp float64) {
	wf.align = true
	wf.alignamp = amp
}

// TODO need to really consider just how a Waveform will interact with underlying data.
// It could possibly act as some kind of pass-through that *looks* like a Sound.
// Maybe that could be done by embedding input?
func (wf *Waveform) Prepare() {
	// don't actually prepare input, input should already be prepared and this should
	// fit into that lifecycle some how.

}

func (wf *Waveform) Paint(ctx gl.Context, sz size.Event) {
	// TODO this is racey and samples can be in the middle of changing
	// move the slice copy to Prepare and sync with playback, or feed over chan
	// TODO assumes mono
	var (
		xstep float32 = 1 / float32(DefaultSampleSize*mixerbuf)
		xpos  float32 = -0.5
	)

	samples := wf.in.Samples()

	if wf.align {
		// naive equivalent-time sampling
		// TODO if audio and graphics were disjoint, a proper equiv-time smpl might be all we really need?
		var mt int
		for i, x := range samples {
			if equals(x, wf.alignamp) {
				mt = i
				break
			}
		}
		for i, x := range samples[mt:] {
			wf.aligned[i] = x
		}
		samples = wf.aligned
	}

	//
	verts := make([]float32, len(samples)*3)
	for i, x := range samples {
		verts[i*3] = float32(xpos)
		verts[i*3+1] = float32(x / 2)
		verts[i*3+2] = 0

		xpos += xstep
	}
	data := f32.Bytes(binary.LittleEndian, verts...)

	//
	ctx.LineWidth(4)

	ctx.UseProgram(wf.program)
	ctx.Uniform4f(wf.color, 1, 1, 1, 1)

	// update hw buf and draw
	ctx.BindBuffer(gl.ARRAY_BUFFER, wf.buf)
	ctx.EnableVertexAttribArray(wf.position)
	ctx.VertexAttribPointer(wf.position, 3, gl.FLOAT, false, 0, 0)
	ctx.BufferSubData(gl.ARRAY_BUFFER, 0, data)
	ctx.DrawArrays(gl.LINE_STRIP, 0, len(samples))
	ctx.DisableVertexAttribArray(wf.position)
}

const vertexShader = `#version 100
attribute vec4 position;
void main() {
  gl_Position = position;
}`

const fragmentShader = `#version 100
precision mediump float;
uniform vec4 color;
void main() {
  gl_FragColor = color;
}`
