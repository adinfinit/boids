package main

import (
	"flag"
	_ "image/png"
	"log"
	"math"
	"math/rand"
	"runtime"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	m "github.com/go-gl/mathgl/mgl32"
)

var (
	windowWidth  = flag.Int("width", 800, "window width")
	windowHeight = flag.Int("height", 600, "window height")
)

const BoidsBatchSize = 1024

type Boids struct {
	VBO uint32

	_first       struct{}
	Position     [BoidsBatchSize]m.Vec3
	Velocity     [BoidsBatchSize]m.Vec3
	_last        struct{}
	Acceleration [BoidsBatchSize]m.Vec3

	Color [BoidsBatchSize]m.Vec3
}

func (boids *Boids) Count() int { return BoidsBatchSize }

func (boids *Boids) size() int {
	return int(unsafe.Offsetof(boids._last) - unsafe.Offsetof(boids._first))
}

func (boids *Boids) Init(program uint32) {
	boids.randomize()

	gl.GenBuffers(1, &boids.VBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, boids.VBO)
	gl.BufferData(gl.ARRAY_BUFFER, boids.size(), unsafe.Pointer(&boids._first), gl.DYNAMIC_DRAW)

	boids.attribVec3(program, "InstancePosition", unsafe.Offsetof(boids.Position))
	boids.attribVec3(program, "InstanceVelocity", unsafe.Offsetof(boids.Velocity))
}

func (boids *Boids) randomize() {
	for i := range boids.Position {
		boids.Position[i] = m.Vec3{
			rand.Float32()*40 - 20,
			rand.Float32()*40 - 20,
			rand.Float32()*40 - 20,
		}
		boids.Velocity[i] = (m.Vec3{
			rand.Float32() - 0.5,
			rand.Float32() - 0.5,
			rand.Float32() - 0.5,
		}).Normalize().Mul(1.5)
	}
}

func (boids *Boids) attribVec3(program uint32, name string, offset uintptr) {
	attrib := uint32(gl.GetAttribLocation(program, gl.Str(name+"\x00")))
	gl.EnableVertexAttribArray(attrib)
	gl.VertexAttribPointer(attrib, 3, gl.FLOAT, false, 3*4, unsafe.Pointer(offset))
	gl.VertexAttribDivisor(attrib, 1)
}

func (boids *Boids) Simulate(world *World) {
	for i, prev := range boids.Position {
		boids.Position[i] = prev.Add(boids.Velocity[i].Mul(world.DeltaTime))
	}
}

func (boids *Boids) Upload() {
	gl.BindBuffer(gl.ARRAY_BUFFER, boids.VBO)
	gl.BufferSubData(gl.ARRAY_BUFFER, 0, boids.size(), unsafe.Pointer(&boids._first))
}

const Mat4Size = 16 * 4

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
	window, err := glfw.CreateWindow(*windowWidth, *windowHeight, "Boids", nil, nil)
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
	program, err := newProgram(vertexShader, fragmentShader, "")
	if err != nil {
		panic(err)
	}

	gl.UseProgram(program)

	timeUniform := gl.GetUniformLocation(program, gl.Str("Time\x00"))
	projectionUniform := gl.GetUniformLocation(program, gl.Str("ProjectionMatrix\x00"))
	cameraUniform := gl.GetUniformLocation(program, gl.Str("CameraMatrix\x00"))
	textureUniform := gl.GetUniformLocation(program, gl.Str("AlbedoTexture\x00"))

	diffuseLightPositionUniform := gl.GetUniformLocation(program, gl.Str("DiffuseLightPosition\x00"))

	diffuseLightLocation := m.Vec3{0, 10, 0}
	gl.Uniform3f(diffuseLightPositionUniform, diffuseLightLocation[0], diffuseLightLocation[1], diffuseLightLocation[2])
	gl.Uniform1i(textureUniform, 0)

	gl.BindFragDataLocation(program, 0, gl.Str("OutputColor\x00"))

	texture, err := LoadTexture("fish.png")
	if err != nil {
		log.Fatalln(err)
	}

	mesh := defaultMesh

	// setup instance data
	var meshVAO uint32
	gl.GenVertexArrays(1, &meshVAO)
	gl.BindVertexArray(meshVAO)

	var meshVBO uint32
	gl.GenBuffers(1, &meshVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, meshVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(mesh.Vertices)*int(MeshVertexBytes), gl.Ptr(mesh.Vertices), gl.STATIC_DRAW)

	meshPositionAttrib := uint32(gl.GetAttribLocation(program, gl.Str("VertexPosition\x00")))
	gl.EnableVertexAttribArray(meshPositionAttrib)
	gl.VertexAttribPointer(meshPositionAttrib, 3, gl.FLOAT, false, MeshVertexBytes, gl.PtrOffset(0))

	meshNormalAttrib := uint32(gl.GetAttribLocation(program, gl.Str("VertexNormal\x00")))
	gl.EnableVertexAttribArray(meshNormalAttrib)
	gl.VertexAttribPointer(meshNormalAttrib, 3, gl.FLOAT, false, MeshVertexBytes, gl.PtrOffset(3*4))

	meshUVAttrib := uint32(gl.GetAttribLocation(program, gl.Str("VertexUV\x00")))
	gl.EnableVertexAttribArray(meshUVAttrib)
	gl.VertexAttribPointer(meshUVAttrib, 2, gl.FLOAT, false, MeshVertexBytes, gl.PtrOffset(3*4+3*4))

	var meshIBO uint32
	gl.GenBuffers(1, &meshIBO)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, meshIBO)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, 2*len(mesh.Indices), gl.Ptr(mesh.Indices), gl.STATIC_DRAW)

	var boids Boids
	boids.Init(program)

	// Configure global settings
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)

	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)

	gl.ClearColor(0x26/255.0, 0x42/255.0, 0x6b/255.0, 1.0)
	log.Println("ERROR: ", gl.GetError())

	angle := float32(0.0)
	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		angle += world.DeltaTime * 0
		sn, cs := math.Sincos(float64(angle))
		world.Camera.Eye[0] = float32(sn) * 10.0
		world.Camera.Eye[2] = float32(cs) * 10.0

		world.NextFrameGLFW(window)

		// Update
		boids.Simulate(world)

		// Render
		gl.UseProgram(program)

		gl.Uniform1f(timeUniform, float32(world.Time))
		gl.UniformMatrix4fv(projectionUniform, 1, false, &world.Camera.Projection[0])
		gl.UniformMatrix4fv(cameraUniform, 1, false, &world.Camera.Camera[0])

		boids.Upload()

		gl.BindVertexArray(meshVAO)

		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, texture.ID)

		gl.DrawElementsInstanced(
			gl.TRIANGLES, int32(len(mesh.Indices)), gl.UNSIGNED_SHORT, gl.PtrOffset(0),
			int32(boids.Count()),
		)

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
	screenSize := m.Vec2{float32(width), float32(height)}
	now := glfw.GetTime()

	if world.ScreenSize != screenSize {
		gl.Viewport(0, 0, int32(width), int32(height))
	}
	world.NextFrame(screenSize, now)
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
		Eye:    m.Vec3{5, 5, 5},
		LookAt: m.Vec3{0, 0, 0},
		Up:     m.Vec3{0, 1, 0},
		FOV:    70,
	}
}

func (camera *Camera) UpdateScreenSize(size m.Vec2) {
	camera.Projection = m.Perspective(m.DegToRad(camera.FOV), size.X()/size.Y(), 0.1, 100.0)
	camera.Camera = m.LookAtV(camera.Eye, camera.LookAt, camera.Up)
}
