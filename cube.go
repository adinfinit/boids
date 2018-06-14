package main

import (
	"math"
	"math/rand"

	m "github.com/go-gl/mathgl/mgl32"
)

type MeshData struct {
	Vertices []float32
	Indices  []int16
}

func (m *MeshData) Vertex(v m.Vec3) int16 {
	p := len(m.Vertices) / 5
	m.Vertices = append(m.Vertices, v[:]...)
	m.Vertices = append(m.Vertices, rand.Float32(), rand.Float32())
	return int16(p)
}

func (m *MeshData) Triangle(a, b, c int16) {
	m.Indices = append(m.Indices, a, b, c)
}

var cube = Lathe(10, 10, true, func(t, phase float32) m.Vec3 {
	r := 12.291*t*t*t - 20*t*t + 8.508*t
	h := 3 * t
	rx := 0.5 * h * float32(math.Exp(float64(1-h)))

	//r := math.Sin(float64(t * math.Pi))
	//r *= float64(t)
	sn, cs := math.Sincos(float64(phase))
	return m.Vec3{
		r * float32(sn) * rx,
		r * float32(cs),
		(t - 0.5) * 3,
	}
})

func impulse(k, x float32) float32 {
	h := k * x
	return h * float32(math.Exp(float64(1-h)))
}

func Lathe(depth, corners int, capped bool, fn func(t, phase float32) m.Vec3) MeshData {
	mesh := MeshData{}

	var headAverage m.Vec3
	lastLayer, nextLayer := make([]int16, corners), make([]int16, corners)
	for pi := 0; pi < corners; pi++ {
		p := float32(pi) * math.Pi * 2 / float32(corners-1)
		v := fn(0, p)
		lastLayer[pi] = mesh.Vertex(v)
		if capped {
			headAverage = headAverage.Add(v)
		}
	}

	if capped {
		headAverage = headAverage.Mul(1 / float32(corners))
		z0 := mesh.Vertex(headAverage)
		for pi := 0; pi < corners; pi++ {
			a, b := lastLayer[pi], lastLayer[(pi+1)%corners]
			mesh.Triangle(z0, a, b)
		}
	}

	var tailAverage m.Vec3
	for ti := 1; ti < depth; ti++ {
		t := float32(ti) / float32(depth-1)
		for pi := 0; pi < corners; pi++ {
			p := float32(pi) * math.Pi * 2 / float32(corners-1)
			v := fn(t, p)
			nextLayer[pi] = mesh.Vertex(v)
			if capped && ti == depth-1 {
				tailAverage = tailAverage.Add(v)
			}
		}

		for pi := 0; pi < corners; pi++ {
			a, b := lastLayer[pi], lastLayer[(pi+1)%corners]
			c, d := nextLayer[pi], nextLayer[(pi+1)%corners]
			mesh.Triangle(a, c, d)
			mesh.Triangle(a, d, b)
		}

		lastLayer, nextLayer = nextLayer, lastLayer
	}

	if capped {
		tailAverage = tailAverage.Mul(1 / float32(corners))
		zt := mesh.Vertex(tailAverage)
		for pi := 0; pi < corners; pi++ {
			a, b := lastLayer[pi], lastLayer[(pi+1)%corners]
			mesh.Triangle(a, zt, b)
		}
	}

	return mesh
}

var cube2 = MeshData{
	Vertices: []float32{
		//  X, Y, Z, U, V
		-1.0, -1.0, -1.0, 0.0, 0.0,
		1.0, -1.0, -1.0, 1.0, 0.0,
		-1.0, -1.0, 1.0, 0.0, 1.0,
		1.0, -1.0, 1.0, 1.0, 1.0,
		-1.0, 1.0, -1.0, 0.0, 0.0,
		-1.0, 1.0, 1.0, 0.0, 1.0,
		1.0, 1.0, -1.0, 1.0, 0.0,
		1.0, 1.0, 1.0, 1.0, 1.0,
		-1.0, -1.0, 1.0, 1.0, 0.0,
		1.0, -1.0, 1.0, 0.0, 0.0,
		-1.0, 1.0, 1.0, 1.0, 1.0,
		1.0, 1.0, 1.0, 0.0, 1.0,
		-1.0, 1.0, -1.0, 0.0, 1.0,
		1.0, 1.0, -1.0, 1.0, 1.0,
		-1.0, 1.0, -1.0, 1.0, 0.0,
		1.0, 1.0, -1.0, 0.0, 0.0,
	},
	Indices: []int16{
		0, 1, 2, 1, 3, 2,
		4, 5, 6, 6, 5, 7,
		8, 9, 10, 9, 11, 10,
		0, 12, 1, 1, 12, 13,
		2, 14, 0, 2, 10, 14,
		3, 1, 15, 3, 15, 11,
	},
}
