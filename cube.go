package main

import (
	"math"
	"unsafe"

	"github.com/adinfinit/g"
)

func impulse(k, x float32) float32 {
	h := k * x
	return h * g.Exp(1-h)
}

var defaultMesh = fish

var sphere = Lathe(12, 12, true, func(t, phase float32) g.Vec3 {
	p := 1 - t*2
	r := -p*p + 1.5
	sn, cs := g.Sincos(phase)
	return g.V3(
		r*sn,
		r*cs,
		(t-0.5)*3,
	)
})

var fish = LatheWrap(12, 12, false, func(t, phase float32) g.Vec3 {
	r := 12.291*t*t*t - 20*t*t + 8.508*t + 0.01
	h := 3 * t
	rx := 0.5*h*g.Exp(1-h) + 0.01

	//r := math.Sin(float64(t * g.Pi))
	//r *= float64(t)
	sn, cs := math.Sincos(float64(phase))
	return g.V3(
		r*float32(sn)*rx,
		r*float32(cs)*0.7,
		(t-0.3)*3,
	)
})

type MeshData struct {
	Vertices []MeshVertex
	Indices  []int16
}

type MeshVertex struct {
	Position g.Vec3
	Normal   g.Vec3
	UV       g.Vec2
}

const MeshVertexBytes = int32(unsafe.Sizeof(MeshVertex{}))

func (mesh *MeshData) Vertex(v g.Vec3, uv g.Vec2) int16 {
	p := len(mesh.Vertices)
	n := g.V3(v.X, v.Y, 0).Normalize()
	mesh.Vertices = append(mesh.Vertices, MeshVertex{
		Position: v,
		Normal:   n,
		UV:       uv,
	})

	return int16(p)
}

func (mesh *MeshData) RecalculateNormals() {
	triangleNormals := make([]g.Vec3, len(mesh.Indices)/3)
	for i := range triangleNormals {
		ai, bi, ci := mesh.Indices[i*3+0], mesh.Indices[i*3+1], mesh.Indices[i*3+2]
		a, b, c := mesh.Vertices[ai].Position, mesh.Vertices[bi].Position, mesh.Vertices[ci].Position
		e1, e2 := b.Sub(a), c.Sub(a)
		triangleNormals[i] = e1.Cross(e2).Normalize()
	}
	triangleCount := make([]int, len(mesh.Vertices))
	for i := range mesh.Vertices {
		mesh.Vertices[i].Normal = g.Z3
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

func (mesh *MeshData) WrapCylinder() {
	z0 := mesh.Vertices[0].Position.Z
	zmin, zmax := z0, z0

	for i := range mesh.Vertices {
		z := mesh.Vertices[i].Position.Z
		if z < zmin {
			zmin = z
		}
		if z > zmax {
			zmax = z
		}
	}

	for i := range mesh.Vertices {
		v := &mesh.Vertices[i]
		p := v.Position
		u := (p.Z - zmin) / (zmax - zmin)
		theta := g.Atan2(p.Y, p.X)/g.Tau + 0.5
		v.UV = g.V2(u, float32(theta))
	}
}

func (mesh *MeshData) Triangle(a, b, c int16) {
	mesh.Indices = append(mesh.Indices, a, b, c)
}

func LatheWrap(depth, corners int, capped bool, fn func(t, phase float32) g.Vec3) MeshData {
	mesh := MeshData{}

	var headAverage g.Vec3
	lastLayer, nextLayer := make([]int16, corners), make([]int16, corners)
	for pi := 0; pi < corners; pi++ {
		p := float32(pi) * g.Tau / float32(corners)
		v := fn(0, p)
		uv := g.V2(0, float32(pi)/float32(corners))
		lastLayer[pi] = mesh.Vertex(v, uv)
		if capped {
			headAverage = headAverage.Add(v)
		}
	}

	if capped {
		headAverage = headAverage.Mul(1 / float32(corners))
		z0 := mesh.Vertex(headAverage, g.V2(0, 0.5))
		for pi := 0; pi < corners; pi++ {
			a, b := lastLayer[pi], lastLayer[(pi+1)%corners]
			mesh.Triangle(z0, a, b)
		}
	}

	var tailAverage g.Vec3
	for ti := 1; ti < depth; ti++ {
		t := float32(ti) / float32(depth-1)
		for pi := 0; pi < corners; pi++ {
			p := float32(pi) * g.Tau / float32(corners)
			v := fn(t, p)
			uv := g.V2(float32(ti)/float32(depth), float32(pi)/float32(corners))
			nextLayer[pi] = mesh.Vertex(v, uv)
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
		uv := g.V2(1, 0.5)
		zt := mesh.Vertex(tailAverage, uv)
		for pi := 0; pi < corners; pi++ {
			a, b := lastLayer[pi], lastLayer[(pi+1)%corners]
			mesh.Triangle(a, zt, b)
		}
	}

	mesh.RecalculateNormals()

	return mesh
}

func Lathe(depth, corners int, capped bool, fn func(t, phase float32) g.Vec3) MeshData {
	mesh := MeshData{}

	var headAverage g.Vec3
	lastLayer, nextLayer := make([]int16, corners+1), make([]int16, corners+1)
	for pi := 0; pi <= corners; pi++ {
		p := float32(pi) * g.Tau / float32(corners)
		v := fn(0, p)
		uv := g.V2(0, float32(pi)/float32(corners))
		lastLayer[pi] = mesh.Vertex(v, uv)
		if capped {
			headAverage = headAverage.Add(v)
		}
	}

	if capped {
		headAverage = headAverage.Mul(1 / float32(corners))
		z0 := mesh.Vertex(headAverage, g.V2(0, 0.5))
		for pi := 0; pi < corners; pi++ {
			a, b := lastLayer[pi], lastLayer[pi+1]
			mesh.Triangle(z0, a, b)
		}
	}

	var tailAverage g.Vec3
	for ti := 1; ti < depth; ti++ {
		t := float32(ti) / float32(depth-1)
		for pi := 0; pi <= corners; pi++ {
			p := float32(pi) * g.Tau / float32(corners)
			v := fn(t, p)
			uv := g.V2(float32(ti)/float32(depth), float32(pi)/float32(corners))
			nextLayer[pi] = mesh.Vertex(v, uv)
			if capped && ti == depth-1 {
				tailAverage = tailAverage.Add(v)
			}
		}

		for pi := 0; pi < corners; pi++ {
			a, b := lastLayer[pi], lastLayer[pi+1]
			c, d := nextLayer[pi], nextLayer[pi+1]
			mesh.Triangle(a, c, d)
			mesh.Triangle(a, d, b)
		}

		lastLayer, nextLayer = nextLayer, lastLayer
	}

	if capped {
		tailAverage = tailAverage.Mul(1 / float32(corners))
		uv := g.V2(1, 0.5)
		zt := mesh.Vertex(tailAverage, uv)
		for pi := 0; pi < corners; pi++ {
			a, b := lastLayer[pi], lastLayer[pi+1]
			mesh.Triangle(a, zt, b)
		}
	}

	mesh.RecalculateNormals()

	return mesh
}
