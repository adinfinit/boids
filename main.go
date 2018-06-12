package main

import (
	"flag"
	_ "image/png"
	"log"
	"runtime"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	m "github.com/go-gl/mathgl/mgl32"
)

var (
	windowWidth  = flag.Int("width", 800, "window width")
	windowHeight = flag.Int("height", 600, "window height")
)

type Boids struct {
	Position     [1024]m.Vec4
	Velocity     [1024]m.Vec4
	Acceleration [1024]m.Vec4
	Color        [1024]m.Vec4
}

func init() { runtime.LockOSThread() }

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	window, err := glfw.CreateWindow(*windowWidth, *windowHeight, "Window Size", nil, nil)
	if err != nil {
		log.Fatalln("failed to create window: ", err)
	}
	window.MakeContextCurrent()

	if err := gl.Init(); err != nil {
		log.Fatalln("failed to initialize glow: ", err)
	}
	log.Println("OpenGL version", gl.GoStr(gl.GetString(gl.VERSION)))

	world := NewWorld()
	world.NextFrameGLFW(window)

	// Configure the vertex and fragment shaders
	program, err := newProgram(vertexShader, fragmentShader)
	if err != nil {
		panic(err)
	}

	gl.UseProgram(program)

	projectionUniform := gl.GetUniformLocation(program, gl.Str("ProjectionMatrix\x00"))
	cameraUniform := gl.GetUniformLocation(program, gl.Str("CameraMatrix\x00"))
	modelUniform := gl.GetUniformLocation(program, gl.Str("ModelMatrix\x00"))
	textureUniform := gl.GetUniformLocation(program, gl.Str("AlbedoTexture\x00"))

	model := m.Ident4()
	gl.UniformMatrix4fv(modelUniform, 1, false, &model[0])
	gl.Uniform1i(textureUniform, 0)

	gl.BindFragDataLocation(program, 0, gl.Str("OutputColor\x00"))

	texture, err := LoadTexture("square.png")
	if err != nil {
		log.Fatalln(err)
	}

	// Configure the vertex data
	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(cubeVertices)*4, gl.Ptr(cubeVertices), gl.STATIC_DRAW)

	vertexPositionAttrib := uint32(gl.GetAttribLocation(program, gl.Str("VertexPosition\x00")))
	gl.EnableVertexAttribArray(vertexPositionAttrib)
	gl.VertexAttribPointer(vertexPositionAttrib, 3, gl.FLOAT, false, 5*4, gl.PtrOffset(0))

	vertexUVAttrib := uint32(gl.GetAttribLocation(program, gl.Str("VertexUV\x00")))
	gl.EnableVertexAttribArray(vertexUVAttrib)
	gl.VertexAttribPointer(vertexUVAttrib, 2, gl.FLOAT, false, 5*4, gl.PtrOffset(3*4))

	var ibo uint32
	gl.GenBuffers(1, &ibo)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ibo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(cubeIndices), gl.Ptr(cubeIndices), gl.STATIC_DRAW)

	var models [16]m.Mat4
	for i := range models {
		models[i] = m.Ident4()
	}
	/*
		var offsetBuffer uint32
		gl.GenBuffers(1, &offsetBuffer)
		gl.BindBuffer(gl.ARRAY_BUFFER, offsetBuffer)
		gl.BufferData(gl.ARRAY_BUFFER, len(models)*16*4, gl.Ptr(models), gl.DYNAMIC_DRAW)

		modelMatrixAttrib := uint32(gl.GetAttribLocation(program, gl.Str("ModelMatrix\x00")))
		gl.EnableVertexAttribArray(modelMatrixAttrib)
		gl.VertexAttribPointer(modelMatrixAttrib, 16, gl.FLOAT, false, 16*4, gl.PtrOffset(0))

		for i := range models {
			models[i] = m.Ident4()
		}
		gl.UniformMatrix4fv(modelUniform, 1, false, &model[0])
		gl.Uniform1i(textureUniform, 0)
		modelUniform := gl.GetUniformLocation(program, gl.Str("ModelMatrix\x00"))
	*/

	// Configure global settings
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.ClearColor(1.0, 1.0, 1.0, 1.0)

	angle := float32(0.0)
	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		world.NextFrameGLFW(window)

		// Update
		angle += world.DeltaTime
		model = m.HomogRotate3D(angle, m.Vec3{0, 1, 0})
		/*
			for i := range models {
				models[i] = m.HomogRotate3D(angle*float32(i)*2.14, m.Vec3{0, 1, 0})
			}
			gl.BufferSubData(offsetBuffer, 0, 16*4, gl.Ptr(models))
		*/

		// Render
		gl.UseProgram(program)
		gl.UniformMatrix4fv(projectionUniform, 1, false, &world.Camera.Projection[0])
		gl.UniformMatrix4fv(cameraUniform, 1, false, &world.Camera.Camera[0])
		gl.UniformMatrix4fv(modelUniform, 1, false, &model[0])

		gl.BindVertexArray(vao)

		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, texture.ID)

		gl.DrawElements(gl.TRIANGLES, int32(len(cubeIndices)), gl.UNSIGNED_BYTE, gl.Ptr(nil))
		// gl.DrawElements(gl.TRIANGLES, 0, 6*2*3)

		// Maintenance
		window.SwapBuffers()
		glfw.PollEvents()
	}
}

type World struct {
	ScreenSize m.Vec2
	Camera     Camera

	Time      float64
	DeltaTime float32
}

func NewWorld() *World {
	world := &World{}
	world.Camera = *NewCamera()
	world.Time = 0
	return world
}

func (world *World) NextFrameGLFW(window *glfw.Window) {
	width, height := window.GetFramebufferSize()
	now := glfw.GetTime()
	world.NextFrame(m.Vec2{float32(width), float32(height)}, now)
}

func (world *World) NextFrame(screenSize m.Vec2, now float64) {
	if world.ScreenSize != screenSize {
		log.Println(screenSize, screenSize.X()/screenSize.Y())
	}
	world.ScreenSize = screenSize
	world.DeltaTime = float32(now - world.Time)
	world.Time = now

	world.Camera.UpdateScreenSize(screenSize)
}

type Camera struct {
	Eye, LookAt, Up m.Vec3

	FOV       float32
	Near, Far float32

	Projection m.Mat4
	Camera     m.Mat4
}

func NewCamera() *Camera {
	return &Camera{
		Eye:    m.Vec3{3, 3, 3},
		LookAt: m.Vec3{0, 0, 0},
		Up:     m.Vec3{0, 1, 0},
		FOV:    45,
	}
}

func (camera *Camera) UpdateScreenSize(size m.Vec2) {
	camera.Projection = m.Perspective(m.DegToRad(camera.FOV), size.X()/size.Y(), 0.1, 10.0)
	camera.Camera = m.LookAtV(camera.Eye, camera.LookAt, camera.Up)
}
