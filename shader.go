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
uniform mat4 ProjectionCameraMatrix;

uniform vec3 DiffuseLightPosition;

in vec3 VertexPosition;
in vec3 VertexNormal;
in vec2 VertexUV;

in vec3  InstancePosition;
in vec3  InstanceHeading;

out vec3 FragmentColor;

const float SWIM_SPEED = 4;
const float SWIM_ROLL_OFFSET = 0.7;
const float SIZE = 0.1;

mat4 LookAt(float size, vec3 pos, vec3 direction) {
	vec3 up = vec3(0, 1, 0);
	vec3 ww = normalize(-direction);
	vec3 uu = normalize(cross(up, ww));
	vec3 vv = normalize(cross(ww, uu));

	// not sure whether correct
	return mat4(
		uu * size,  0,
		vv * size,  0,
		ww * size,  0,
		pos, 1
	);
}

mat4 LookAtOptimized(float size, vec3 pos, vec3 direction) {
	vec3 ww = -direction;
	vec3 uu = normalize(vec3(ww.z, 0, -ww.x));
	vec3 vv = normalize(vec3(ww.y*uu.z, ww.z*uu.x - ww.x*uu.z, -ww.y*uu.x));

	// not sure whether correct
	return mat4(
		uu * size,  0,
		vv * size,  0,
		ww * size,  0,
		pos, 1
	);
}

vec3 RotateZ(vec3 original, vec2 twistRotation) {
	float sn, cs;
	sn = twistRotation.x;
	cs = twistRotation.y;

	vec3 r = original;
	r.x = original.x * cs - original.y * sn;
	r.y = original.x * sn + original.y * cs;
	return r;
}

vec3 Swim(vec3 original, vec2 twistRotation, float wiggleAmount) {
	original = RotateZ(original, twistRotation);
	vec3 result = original;
	result.x += wiggleAmount;
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
	
	mat4 modelMatrix = LookAtOptimized(SIZE, InstancePosition, InstanceHeading);
	mat4 normalMatrix = transpose(inverse(CameraMatrix * modelMatrix));

	float twistAmount = sin(-VertexPosition.z + phase + Time * SWIM_SPEED - SWIM_ROLL_OFFSET)*0.3;
	float wiggleAmount = sin(Time * SWIM_SPEED - VertexPosition.z + phase) * 0.2;
	vec2 twistRotation = vec2(sin(twistAmount), cos(twistAmount));

	vec3 position = Swim(VertexPosition, twistRotation, wiggleAmount);
	vec3 normal = normalize(Swim(VertexPosition + VertexNormal, twistRotation, wiggleAmount) - position);

	vec4 fragmentPosition = modelMatrix * vec4(position, 1);
	gl_Position = ProjectionCameraMatrix * fragmentPosition;

	// lighting
	float hue = mod(gl_InstanceID * 0.011111, 1);
	vec3 albedo = vec3(hue);
	float ambientLight = 0.3;
	
	vec3 screenNormal = normalize(mat3(normalMatrix) * normal);
	vec3 diffuseLightDirection = normalize(DiffuseLightPosition - fragmentPosition.xyz);
	float diffuseShade = clamp(dot(screenNormal, diffuseLightDirection), 0.0, 1.0);

	FragmentColor = albedo * (ambientLight + diffuseShade);
}
` + "\x00"

var fragmentShader = `
#version 330

in  vec3 FragmentColor;
out vec4 OutputColor;

void main() {
	OutputColor = vec4(FragmentColor, 1);
}
` + "\x00"
