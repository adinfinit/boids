package main

import (
	"fmt"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
)

func newProgram(vertexShaderSource, fragmentShaderSource string) (uint32, error) {
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}

	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	program := gl.CreateProgram()

	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	return program, nil
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}

var vertexShader = `
#version 330

uniform mat4 ProjectionMatrix;
uniform mat4 CameraMatrix;

in vec3 VertexPosition;
in vec3 VertexNormal;
in vec2 VertexUV;

in mat4 ModelMatrix;

out vec3 FragmentPosition;
out vec3 FragmentNormal;
out vec2 FragmentUV;

void main() {
	FragmentUV = VertexUV;
	FragmentNormal = normalize(mat3(ModelMatrix) * VertexNormal);

	vec4 fragmentPosition = ModelMatrix * vec4(VertexPosition, 1);
	FragmentPosition = fragmentPosition.xyz;
    gl_Position = ProjectionMatrix * CameraMatrix * fragmentPosition;
}
` + "\x00"

var fragmentShader = `
#version 330

uniform sampler2D AlbedoTexture;

uniform vec3 AmbientLightLocation;

in vec3 FragmentPosition;
in vec3 FragmentNormal;
in vec2 FragmentUV;

out vec4 OutputColor;

void main() {
	vec3 location = normalize(AmbientLightLocation - FragmentPosition);
	float ambientShade = clamp(dot(FragmentNormal, location), 0.2, 1);

	vec2 uv = FragmentUV;
    OutputColor = texture(AlbedoTexture, uv) * ambientShade;
}
` + "\x00"
