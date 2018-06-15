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

uniform float Time;

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
	vec3 position = VertexPosition;
	float phase = mod(gl_InstanceID, 3.14);
	position.x += sin(Time * 4 - position.z + phase) * position.z;
	
	FragmentUV = VertexUV;
	//FragmentNormal = mat3(ModelMatrix) * VertexNormal;
	//FragmentNormal = mat3(inverse(ModelMatrix)) * VertexNormal;
	FragmentNormal = mat3(transpose(inverse(ModelMatrix))) * VertexNormal;
	
	FragmentPosition = (ModelMatrix * vec4(position, 1)).xyz;
    gl_Position = ProjectionMatrix * CameraMatrix * ModelMatrix * vec4(position, 1);
}
` + "\x00"

var fragmentShader = `
#version 330

uniform sampler2D AlbedoTexture;

uniform vec3 DiffuseLightPosition;

in vec3 FragmentPosition;
in vec3 FragmentNormal;
in vec2 FragmentUV;

out vec4 OutputColor;

void main() {
	float ambientLight = 0.1;
	
	vec3 normal = normalize(FragmentNormal);
	vec3 diffuseLightDirection = normalize(DiffuseLightPosition - FragmentPosition);

	float diffuseShade = clamp(dot(normal, diffuseLightDirection), 0.0, 1.0);

	vec2 uv = FragmentUV;
    OutputColor = texture(AlbedoTexture, uv) * (ambientLight + diffuseShade);
}
` + "\x00"
