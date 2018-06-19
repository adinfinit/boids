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

in vec3  InstancePosition;
in vec3  InstanceHeading;
in float InstanceHue;

out vec3 FragmentPosition;
out vec3 FragmentNormal;
out vec2 FragmentUV;
out vec4 FragmentColor;

out float FragmentOrient;

#define SWIM_SPEED 4
#define SWIM_ROLL_OFFSET 0.7
#define SIZE 0.2

mat4 scale(float size) {
	return mat4(
		size, 0, 0, 0,
		0, size, 0, 0,
		0, 0, size, 0,
		0, 0, 0, 1
	);
}

mat4 Translate(vec3 pos) {
	return mat4(
		SIZE, 0, 0, 0,
		0, SIZE, 0, 0,
		0, 0, SIZE, 0,
		pos, 1
	);
}

mat4 LookAt(float size, vec3 pos, vec3 direction) {
	vec3 up = vec3(0, 1, 0);

	vec3 ww = normalize(-direction);
	vec3 uu = normalize(cross(up, ww));
	vec3 vv = normalize(cross(ww, uu));

	// not sure whether correct
	return mat4(
		uu,  0,
		vv,  0,
		ww,  0,
		pos, 1
	) * scale(SIZE);
}

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
	result.x += sin(Time * SWIM_SPEED - original.z + phase) * 0.5;
	return result;
}

vec3 hsv2rgb(vec3 c)
{
    vec4 K = vec4(1.0, 2.0 / 3.0, 1.0 / 3.0, 3.0);
    vec3 p = abs(fract(c.xxx + K.xyz) * 6.0 - K.www);
    return c.z * mix(K.xxx, clamp(p - K.xxx, 0.0, 1.0), c.y);
}

void main() {
	float phase = mod(gl_InstanceID, 3.14);
	
	mat4 modelMatrix = LookAt(SIZE, InstancePosition, InstanceHeading);
	mat4 normalMatrix = transpose(inverse(CameraMatrix * modelMatrix));

	// vec3 position = VertexPosition;
	// vec3 normal = VertexNormal;
	vec3 position = Swim(VertexPosition, phase);
	vec3 normal = normalize(Swim(VertexPosition + VertexNormal, phase) - position);
	
	FragmentUV = VertexUV;
	FragmentNormal = normalize(mat3(normalMatrix) * normal);
	
	gl_Position = ProjectionMatrix * CameraMatrix * modelMatrix * vec4(position, 1);

	FragmentPosition = vec3(modelMatrix * vec4(position, 1));
	FragmentColor = vec4(hsv2rgb(vec3(InstanceHue, 0.8, 0.7)), 1);
}
` + "\x00"

var fragmentShader = `
#version 330

const float TAU = 2.0 * 3.14;

const float IRI_NOISE_INTENSITY        = 4.0;
const float IRI_GAMMA = 0.9;
const float IRI_CURVE = 0.0;

const vec3  IRI_ORIENTATION_FREQUENCY  = vec3(1.0, 1.0, 1.0);
const vec3  IRI_ORIENTATION_OFFSET     = vec3(0.0, 0.0, 0.0);
const vec3  IRI_NOISE_FREQUENCY        = vec3(1.0, 1.0, 1.0);
const vec3  IRI_NOISE_OFFSET           = vec3(0.0, 0.0, 0.0);

uniform sampler2D AlbedoTexture;

uniform vec3 DiffuseLightPosition;

in vec3 FragmentPosition;
in vec3 FragmentNormal;
in vec2 FragmentUV;
in vec4 FragmentColor;

out vec4 OutputColor;

void main() {
	float ambientLight = 0.1;
	
	vec3 normal = normalize(FragmentNormal);
	vec3 diffuseLightDirection = normalize(DiffuseLightPosition - FragmentPosition);

	float diffuseShade = clamp(dot(normal, diffuseLightDirection), 0.0, 1.0);

	vec4 albedo = texture(AlbedoTexture, FragmentUV);

	OutputColor = albedo * (ambientLight + diffuseShade) * FragmentColor;
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
