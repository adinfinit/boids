package main

import (
	"math"
	"unsafe"

	m "github.com/go-gl/mathgl/mgl32"
)

func impulse(k, x float32) float32 {
	h := k * x
	return h * float32(math.Exp(float64(1-h)))
}

var cube = Lathe(12, 12, true, func(t, phase float32) m.Vec3 {
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

type MeshData struct {
	Vertices []MeshVertex
	Indices  []int16
}

type MeshVertex struct {
	Position m.Vec3
	Normal   m.Vec3
	UV       m.Vec2
}

const MeshVertexBytes = int32(unsafe.Sizeof(MeshVertex{}))

func (mesh *MeshData) Vertex(v m.Vec3) int16 {
	p := len(mesh.Vertices)
	n := (m.Vec3{v[0], v[1], 0}).Normalize()

	theta := float32(math.Atan2(float64(v.Y()), float64(v.X())))
	pt := theta*0.5/math.Pi + 0.5
	uv := m.Vec2{v.Z()/3 + 0.4, pt}

	mesh.Vertices = append(mesh.Vertices, MeshVertex{
		Position: v,
		Normal:   n,
		UV:       uv,
	})

	return int16(p)
}

func (mesh *MeshData) RecalculateNormals() {
	triangleNormals := make([]m.Vec3, len(mesh.Indices)/3)
	for i := range triangleNormals {
		ai, bi, ci := mesh.Indices[i*3+0], mesh.Indices[i*3+1], mesh.Indices[i*3+2]
		a, b, c := mesh.Vertices[ai].Position, mesh.Vertices[bi].Position, mesh.Vertices[ci].Position
		e1, e2 := b.Sub(a), c.Sub(a)
		triangleNormals[i] = e1.Cross(e2).Normalize()
	}
	triangleCount := make([]int, len(mesh.Vertices))
	for i := range mesh.Vertices {
		mesh.Vertices[i].Normal = m.Vec3{}
	}
	for i, mi := range mesh.Indices {
		trin := triangleNormals[i/3]
		v := &mesh.Vertices[mi]
		v.Normal = v.Normal.Add(trin)
		triangleCount[mi]++
	}
	for i := range mesh.Vertices {
		v := &mesh.Vertices[i]
		v.Normal.Mul(1 / float32(triangleCount[i]))
	}
}

func (mesh *MeshData) Triangle(a, b, c int16) {
	mesh.Indices = append(mesh.Indices, a, b, c)
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

	mesh.RecalculateNormals()

	return mesh
}
