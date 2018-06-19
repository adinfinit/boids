// +build ignore

package main

import (
	"github.com/adinfinit/g"
	"github.com/go-gl/gl/v4.1-core/gl"
)

type Program struct {
	ID uint32

	VertexPositionLoc uint32

	ModelPositionLoc  uint32
	ModelDirectionLoc uint32
}

type Mesh struct {
	Program uint32

	VAO uint32
	VBO uint32
	IBO uint32

	Vertices []Vertex
	Indicies []uint16
}

type Vertex struct {
	Position g.Vec3
}

func (mesh *Mesh) Upload() error {
	// setup instance data
	gl.GenVertexArrays(1, &mesh.VAO)
	gl.BindVertexArray(mesh.VAO)

	gl.GenBuffers(1, &mesh.VBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, mesh.VBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(mesh.Vertices)*4, gl.Ptr(mesh.Vertices), gl.STATIC_DRAW)

	meshPositionAttrib := uint32(gl.GetAttribLocation(program, gl.Str("VertexPosition\x00")))
	gl.EnableVertexAttribArray(meshPositionAttrib)
	gl.VertexAttribPointer(meshPositionAttrib, 3, gl.FLOAT, false, 5*4, gl.PtrOffset(0))

	gl.GenBuffers(1, &mesh.IBO)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, mesh.IBO)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, 2*len(mesh.Indicies), gl.Ptr(mesh.Indicies), gl.STATIC_DRAW)
}
