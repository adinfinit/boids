// +build ignore

package main

import (
	"github.com/adinfinit/g"
	"github.com/go-gl/gl/v4.1-core/gl"
)

type Shader struct {
	program       uint32
	locationCache map[string]int32
}

func NewShader() *Shader {
	return &Shader{}
}

func (shader *Shader) Recompile() {}

func (shader *Shader) Begin() { gl.UseProgram(shader.program) }
func (shader *Shader) End()   { gl.UseProgram(0) }

func (shader *Shader) uniformLocation(name string) int32 {
	location, ok := shader.locationCache[name]
	if !ok {
		location = gl.GetUniformLocation(shader.program, gl.Str(name+"\x00"))
		shader.locationCache[name] = location
	}
	return location
}

func (shader *Shader) UniformFloat32(name string, v float32) {
	location := shader.uniformLocation(name)
	if location == 0 {
		return
	}
	gl.Uniform1f(location, v)
}

func (shader *Shader) UniformVec3(name string, v g.Vec3) {
	location := shader.uniformLocation(name)
	if location == 0 {
		return
	}
	gl.Uniform3f(location, v.X, v.Y, v.Z)
}

func (shader *Shader) UniformMatrix(name string, v g.Mat4) {
	location := shader.uniformLocation(name)
	if location == 0 {
		return
	}
	gl.UniformMatrix4fv(location, 1, false, v.Ptr())
}
