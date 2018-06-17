package main

import (
	"fmt"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
)

func newProgram(vertexShaderSource, fragmentShaderSource, geometryShaderSource string) (uint32, error) {
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
	defer gl.DeleteShader(vertexShader)

	gl.AttachShader(program, fragmentShader)
	defer gl.DeleteShader(fragmentShader)

	if geometryShaderSource != "" {
		geometryShader, err := compileShader(geometryShaderSource, gl.GEOMETRY_SHADER)
		if err != nil {
			return 0, err
		}
		gl.AttachShader(program, geometryShader)
		defer gl.DeleteShader(geometryShader)
	}

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

in vec3 InstancePosition;
in vec3 InstanceVelocity;

out vec3 FragmentPosition;
out vec3 FragmentNormal;
out vec2 FragmentUV;

#define SWIM_SPEED 4
#define SWIM_ROLL_OFFSET 1
#define SIZE 0.25

vec3 RotateZ(vec3 original, float alpha) {
	float sn, cs;
	sn = sin(alpha);
	cs = cos(alpha);
	mat2 m = mat2(cs, -sn, sn, cs);
	return vec3(m * original.xy, original.z);
}

vec3 Swim(vec3 original, float phase) {
	original = RotateZ(original, sin(-original.z + phase + Time * SWIM_SPEED - SWIM_ROLL_OFFSET)*0.5);
	vec3 result = original;
	result.x += sin(Time * SWIM_SPEED - original.z + phase) * original.z;
	return result;
}

void main() {
	float phase = mod(gl_InstanceID, 3.14);

	mat4 modelMatrix = mat4(
		SIZE, 0, 0, 0,
		0, SIZE, 0, 0,
		0, 0, SIZE, 0,
		InstancePosition.x, InstancePosition.y, InstancePosition.z, 1
	);

	mat4 modelMatrixInvT = mat4(
		1.0/SIZE, 0, 0, -InstancePosition.x,
		0, 1.0/SIZE, 0, -InstancePosition.y,
		0, 0, 1.0/SIZE, -InstancePosition.z,
		0, 0, 0, 1
	);

	vec3 position = Swim(VertexPosition, phase);
	vec3 normal = Swim(VertexPosition + VertexNormal * 0.1, phase) - position;
	normal = normalize(normal);
	
	FragmentUV = VertexUV;
	FragmentNormal = mat3(modelMatrixInvT) * normal;
	
	FragmentPosition = (modelMatrix * vec4(position, 1)).xyz;
	gl_Position = ProjectionMatrix * CameraMatrix * modelMatrix * vec4(position, 1);
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

var normalVertexShader = `
#version 330

uniform float Time;

uniform mat4 ProjectionMatrix;
uniform mat4 CameraMatrix;

in vec3 VertexPosition;
in vec3 VertexNormal;
in vec2 VertexUV;


out VS_OUT {
	vec3 FragmentNormal;
} vs_out;

#define SWIM_SPEED 4
#define SWIM_ROLL_OFFSET 1

vec3 RotateZ(vec3 original, float alpha) {
	float sn, cs;
	sn = sin(alpha);
	cs = cos(alpha);
	mat2 m = mat2(cs, -sn, sn, cs);
	return vec3(m * original.xy, original.z);
}

vec3 Swim(vec3 original, float phase) {
	original = RotateZ(original, sin(-original.z + phase + Time * SWIM_SPEED - SWIM_ROLL_OFFSET)*0.5);
	vec3 result = original;
	result.x += sin(Time * SWIM_SPEED - original.z + phase) * original.z;
	return result;
}

void main() {
	float phase = mod(gl_InstanceID, 3.14);

	vec3 position = Swim(VertexPosition, phase);
	vec3 normal = Swim(VertexPosition + VertexNormal * 0.1, phase) - position;
	normal = normalize(normal);
	
	vs_out.FragmentNormal = normal;
	gl_Position = ProjectionMatrix * CameraMatrix * ModelMatrix * vec4(position, 1);
}
` + "\x00"

var normalGeometryShader = `
#version 330 core
layout (triangles) in;
layout (line_strip, max_vertices = 6) out;

in VS_OUT {
    vec3 FragmentNormal;
} gs_in[];

const float MAGNITUDE = 0.2;

void GenerateLine(int index)
{
    gl_Position = gl_in[index].gl_Position;
    EmitVertex();
    gl_Position = gl_in[index].gl_Position + vec4(gs_in[index].FragmentNormal, 0.0) * MAGNITUDE;
    EmitVertex();
    EndPrimitive();
}

void main()
{
    GenerateLine(0);
    GenerateLine(1);
    GenerateLine(2);
}
` + "\x00"

var normalFragmentShader = `
#version 330

out vec4 OutputColor;

void main() {
	OutputColor = vec4(1.0, 1.0, 0.0, 1.0);
}
` + "\x00"
