package main

import (
	"flag"
	"fmt"
	_ "image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"unsafe"

	"github.com/adinfinit/g"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"

	"github.com/egonelbre/async"
	"github.com/loov/hrtime"
)

var (
	cpuprofile = flag.String("cpuprofile", "", "profile")

	windowWidth  = flag.Int("width", 800, "window width")
	windowHeight = flag.Int("height", 600, "window height")

	procs = flag.Int("p", runtime.GOMAXPROCS(-1), "parallelism")
)

const (
	BoidsBatchSize = 100000
)

type Boids struct {
	VBO uint32

	Settings struct {
		CellRadius       float32
		SeparationWeight float32
		AlignmentWeight  float32
		TargetWeight     float32
	}

	*GPUBoids

	Speed     [BoidsBatchSize]float32
	CellIndex [BoidsBatchSize]int32

	Targets []g.Vec3

	CellHash       map[int32][]int32
	CellIndices    [][]int32
	CellTarget     []g.Vec3
	CellAlignment  []g.Vec3
	CellSeparation []g.Vec3
}

type GPUBoids struct {
	Position [BoidsBatchSize]g.Vec3
	Heading  [BoidsBatchSize]g.Vec3
}

func (boids *Boids) randomize() {
	for i := range boids.Position {
		boids.Position[i] = g.V3(
			rand.Float32()*40-20,
			rand.Float32()*40-20,
			rand.Float32()*40-20,
		)
		boids.Heading[i] = g.V3(
			rand.Float32()-0.5,
			rand.Float32()-0.5,
			rand.Float32()-0.5,
		).Normalize()
		boids.Speed[i] = 5
	}
}

func (boids *Boids) initData() {
	boids.GPUBoids = &GPUBoids{}
	boids.CellHash = make(map[int32][]int32, BoidsBatchSize/10)

	boids.Settings.CellRadius = 5
	boids.Settings.SeparationWeight = 0.5
	boids.Settings.AlignmentWeight = 1
	boids.Settings.TargetWeight = 0.5

	boids.Targets = []g.Vec3{{}, {}, {}}
}

var frame int

func bench(name string) func() {
	start := hrtime.TSC()
	return func() {
		stop := hrtime.TSC()
		if frame%100 == 0 {
			fmt.Printf("%-15s: %v\n", name, (stop - start).ApproxDuration())
		}
	}
}

func (boids *Boids) Simulate(world *World) {
	frame++

	sn, cs := math.Sincos(float64(world.Time * 0.1))

	boids.Targets[0] = g.V3(
		0,
		float32(cs)*20,
		float32(sn)*20,
	)

	boids.Targets[1] = g.V3(
		float32(sn)*25,
		0,
		float32(cs)*25,
	)

	boids.Targets[2] = g.V3(
		-float32(cs)*30,
		float32(sn)*30,
		0,
	)

	boids.Settings.TargetWeight = 1

	for hash, list := range boids.CellHash {
		boids.CellHash[hash] = list[:0]
	}

	bench("---")()
	boids.hashPositions(boids.Settings.CellRadius)
	boids.resizeCells()
	boids.computeCells(world)
	boids.steerAndMove(world)
}

func (boids *Boids) hashPositions(radius float32) {
	defer bench("hashPositions")()

	invradius := 1 / radius
	for i, p := range boids.Position {
		x, y, z := int32(p.X*invradius), int32(p.Y*invradius), int32(p.Z*invradius)

		hash := x
		hash += (hash * 397) ^ y
		hash += (hash * 397) ^ z
		hash += hash << 3
		hash ^= hash >> 11
		hash += hash << 15

		boids.CellHash[hash] = append(boids.CellHash[hash], int32(i))
	}
}

func (boids *Boids) resizeCells() {
	defer bench("resizeCells")()
	if cap(boids.CellAlignment) < len(boids.CellHash) {
		boids.CellAlignment = make([]g.Vec3, len(boids.CellHash))
		boids.CellSeparation = make([]g.Vec3, len(boids.CellHash))
		boids.CellTarget = make([]g.Vec3, len(boids.CellHash))
		boids.CellIndices = make([][]int32, len(boids.CellHash))
	}

	boids.CellAlignment = boids.CellAlignment[:len(boids.CellHash)]
	boids.CellSeparation = boids.CellSeparation[:len(boids.CellHash)]
	boids.CellTarget = boids.CellTarget[:len(boids.CellHash)]
	boids.CellIndices = boids.CellIndices[:len(boids.CellHash)]

	nextIndex := 0
	for _, indices := range boids.CellHash {
		boids.CellIndices[nextIndex] = indices
		nextIndex++
	}
}

func (boids *Boids) computeCells(world *World) {
	defer bench("computeCells")()

	async.Iter(len(boids.CellIndices), *procs, func(cellIndex int) {
		if len(boids.CellIndices[cellIndex]) == 0 {
			return
		}

		indices := boids.CellIndices[cellIndex]

		alignment := g.Vec3{}
		separation := g.Vec3{}

		for _, boidIndex := range indices {
			boids.CellIndex[boidIndex] = int32(cellIndex)
			alignment = alignment.Add(boids.Heading[boidIndex])
			separation = separation.Add(boids.Position[boidIndex])
		}

		byCount := 1.0 / float32(len(indices))
		center := separation.Mul(byCount)
		boids.CellAlignment[cellIndex] = alignment.Mul(byCount)
		boids.CellSeparation[cellIndex] = center

		nearest := boids.Targets[0]
		nearestDistance2 := center.Sub(boids.Targets[0]).Len2()
		for _, target := range boids.Targets[1:] {
			dist2 := center.Sub(target).Len2()
			if dist2 < nearestDistance2 {
				nearest = target
				nearestDistance2 = dist2
			}
		}
		boids.CellTarget[cellIndex] = nearest
	})
}

func check(t string, ps ...g.Vec3) {
	for i, p := range ps {
		if math.IsNaN(float64(p.X)) || math.IsNaN(float64(p.Y)) || math.IsNaN(float64(p.Z)) {
			fmt.Println(ps)
			panic(t + "=" + strconv.Itoa(i))
		}
	}
}

func safeNormalize(v g.Vec3, s float32) g.Vec3 {
	l := v.Len2()
	if l < 1e-3 {
		return g.V3(0, 0, s)
	}
	return v.Mul(s / g.Sqrt(l))
}

func (boids *Boids) steerAndMove(world *World) {
	defer bench("steerAndMove")()
	dt := world.DeltaTime

	async.BlockIter(len(boids.Position), *procs, func(start, limit int) {
		for offset := range boids.Position[start:limit] {
			i := start + offset
			cell := boids.CellIndex[i]
			pos := boids.Position[i]
			head := boids.Heading[i]

			cellSeparation := boids.CellSeparation[cell]
			cellAlignment := boids.CellAlignment[cell]
			cellTarget := boids.CellTarget[cell]

			separation := safeNormalize(pos.Sub(cellSeparation), boids.Settings.SeparationWeight)
			target := safeNormalize(cellTarget.Sub(pos), boids.Settings.TargetWeight)
			alignment := safeNormalize(cellAlignment.Sub(head), boids.Settings.AlignmentWeight)

			normalHeading := safeNormalize(alignment.Add(separation).Add(target), 1)
			newHeading := safeNormalize(head.Add(normalHeading.Sub(head).Mul(dt)), 1)
			boids.Heading[i] = newHeading

			boids.Position[i] = boids.Position[i].Add(newHeading.Mul(dt * boids.Speed[i]))
		}
	})
}

func (boids *Boids) Count() int { return BoidsBatchSize }

func (boids *Boids) size() int {
	return int(unsafe.Sizeof(*boids.GPUBoids))
}
func (boids *Boids) Init(program uint32) {
	boids.initData()
	boids.randomize()

	gl.GenBuffers(1, &boids.VBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, boids.VBO)
	gl.BufferData(gl.ARRAY_BUFFER, boids.size(), unsafe.Pointer(boids.GPUBoids), gl.DYNAMIC_DRAW)

	boids.attribVec3(program, "InstancePosition", unsafe.Offsetof(boids.GPUBoids.Position))
	boids.attribVec3(program, "InstanceHeading", unsafe.Offsetof(boids.GPUBoids.Heading))
}

func (boids *Boids) attribVec3(program uint32, name string, offset uintptr) {
	attrib := uint32(gl.GetAttribLocation(program, gl.Str(name+"\x00")))
	gl.EnableVertexAttribArray(attrib)
	gl.VertexAttribPointer(attrib, 3, gl.FLOAT, false, 3*4, unsafe.Pointer(offset))
	gl.VertexAttribDivisor(attrib, 1)
}

func (boids *Boids) attribRGBA8(program uint32, name string, offset uintptr) {
	attrib := uint32(gl.GetAttribLocation(program, gl.Str(name+"\x00")))
	gl.EnableVertexAttribArray(attrib)
	gl.VertexAttribPointer(attrib, 4, gl.UNSIGNED_BYTE, true, 4, unsafe.Pointer(offset))
	gl.VertexAttribDivisor(attrib, 1)
}

func (boids *Boids) attribFloat(program uint32, name string, offset uintptr) {
	attrib := uint32(gl.GetAttribLocation(program, gl.Str(name+"\x00")))
	gl.EnableVertexAttribArray(attrib)
	gl.VertexAttribPointer(attrib, 1, gl.FLOAT, false, 4, unsafe.Pointer(offset))
	gl.VertexAttribDivisor(attrib, 1)
}

func (boids *Boids) Upload() {
	gl.BindBuffer(gl.ARRAY_BUFFER, boids.VBO)
	gl.BufferSubData(gl.ARRAY_BUFFER, 0, boids.size(), unsafe.Pointer(boids.GPUBoids))
}

const Mat4Size = 16 * 4

func init() { runtime.LockOSThread() }

func main() {
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatalf("unable to create cpu-profile %q: %v", *cpuprofile, err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("unable to start cpu-profile: %v", err)
		}
		defer pprof.StopCPUProfile()
	}

	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.Samples, 2)

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
	boidProgram, err := newProgram(vertexShader, fragmentShader, "")
	if err != nil {
		panic(err)
	}

	gl.UseProgram(boidProgram)

	timeUniform := gl.GetUniformLocation(boidProgram, gl.Str("Time\x00"))
	projectionUniform := gl.GetUniformLocation(boidProgram, gl.Str("ProjectionMatrix\x00"))
	cameraUniform := gl.GetUniformLocation(boidProgram, gl.Str("CameraMatrix\x00"))
	projectionCameraUniform := gl.GetUniformLocation(boidProgram, gl.Str("ProjectionCameraMatrix\x00"))

	diffuseLightPositionUniform := gl.GetUniformLocation(boidProgram, gl.Str("DiffuseLightPosition\x00"))

	gl.BindFragDataLocation(boidProgram, 0, gl.Str("OutputColor\x00"))

	mesh := defaultMesh

	// setup instance data
	var meshVAO uint32
	gl.GenVertexArrays(1, &meshVAO)
	gl.BindVertexArray(meshVAO)

	var meshVBO uint32
	gl.GenBuffers(1, &meshVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, meshVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(mesh.Vertices)*int(MeshVertexBytes), gl.Ptr(mesh.Vertices), gl.STATIC_DRAW)

	meshPositionAttrib := uint32(gl.GetAttribLocation(boidProgram, gl.Str("VertexPosition\x00")))
	gl.EnableVertexAttribArray(meshPositionAttrib)
	gl.VertexAttribPointer(meshPositionAttrib, 3, gl.FLOAT, false, MeshVertexBytes, gl.PtrOffset(0))

	meshNormalAttrib := uint32(gl.GetAttribLocation(boidProgram, gl.Str("VertexNormal\x00")))
	gl.EnableVertexAttribArray(meshNormalAttrib)
	gl.VertexAttribPointer(meshNormalAttrib, 3, gl.FLOAT, false, MeshVertexBytes, gl.PtrOffset(3*4))

	meshUVAttrib := uint32(gl.GetAttribLocation(boidProgram, gl.Str("VertexUV\x00")))
	gl.EnableVertexAttribArray(meshUVAttrib)
	gl.VertexAttribPointer(meshUVAttrib, 2, gl.FLOAT, false, MeshVertexBytes, gl.PtrOffset(3*4+3*4))

	var meshIBO uint32
	gl.GenBuffers(1, &meshIBO)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, meshIBO)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, 2*len(mesh.Indices), gl.Ptr(mesh.Indices), gl.STATIC_DRAW)

	boids := &Boids{}
	boids.Init(boidProgram)

	// Configure global settings
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)

	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)

	// gl.ClearColor(0x26/255.0, 0x42/255.0, 0x6b/255.0, 1.0)
	gl.ClearColor(0, 0, 0, 1.0)
	log.Println("ERROR: ", gl.GetError())

	angle := float32(0.0)
	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		angle += world.DeltaTime * 0
		sn, cs := g.Sincos(angle)
		world.Camera.Eye.X = sn * 30.0
		world.Camera.Eye.Z = cs * 30.0
		world.DiffuseLightPosition = g.Z3

		world.NextFrameGLFW(window)

		// Update
		simStart := hrtime.Now()
		boids.Simulate(world)
		simStop := hrtime.Now()

		// Rendering
		renderStart := hrtime.Now()
		boids.Upload()

		gl.UseProgram(boidProgram)

		gl.Uniform1f(timeUniform, float32(world.Time))
		gl.UniformMatrix4fv(projectionUniform, 1, false, world.Camera.Projection.Ptr())
		gl.UniformMatrix4fv(cameraUniform, 1, false, world.Camera.Camera.Ptr())
		gl.UniformMatrix4fv(projectionCameraUniform, 1, false, world.Camera.ProjectionCamera.Ptr())
		gl.Uniform3fv(diffuseLightPositionUniform, 1, world.DiffuseLightPosition.Ptr())

		gl.BindVertexArray(meshVAO)

		gl.DrawElementsInstanced(
			gl.TRIANGLES, int32(len(mesh.Indices)), gl.UNSIGNED_SHORT, gl.PtrOffset(0),
			int32(boids.Count()),
		)
		// gl.Finish()

		renderStop := hrtime.Now()

		window.SetTitle(fmt.Sprintf("Sim:%v    Render:%v", simStop-simStart, renderStop-renderStart))

		// Maintenance
		window.SwapBuffers()
		glfw.PollEvents()
	}
}

type World struct {
	ScreenSize g.Vec2
	Camera     Camera

	DiffuseLightPosition g.Vec3

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
	screenSize := g.V2(float32(width), float32(height))
	now := glfw.GetTime()

	if world.ScreenSize != screenSize {
		gl.Viewport(0, 0, int32(width), int32(height))
	}
	world.NextFrame(screenSize, now)
}

func (world *World) NextFrame(screenSize g.Vec2, now float64) {
	if world.ScreenSize != screenSize {
		log.Println(screenSize, screenSize.X/screenSize.Y)
	}
	world.ScreenSize = screenSize
	world.DeltaTime = float32(now - world.Time)
	world.Time = now

	world.Camera.UpdateScreenSize(screenSize)
}

type Camera struct {
	Eye, LookAt, Up g.Vec3

	FOV       float32
	Near, Far float32

	Projection       g.Mat4
	Camera           g.Mat4
	ProjectionCamera g.Mat4
}

func NewCamera() *Camera {
	return &Camera{
		Eye:    g.V3(30, 30, 30),
		LookAt: g.V3(0, 0, 0),
		Up:     g.V3(0, 1, 0),
		FOV:    70,
	}
}

func (camera *Camera) UpdateScreenSize(size g.Vec2) {
	camera.Projection = g.Perspective(g.DegToRad(camera.FOV), size.X/size.Y, 0.1, 100.0)
	camera.Camera = g.LookAtV(camera.Eye, camera.LookAt, camera.Up)

	camera.ProjectionCamera = camera.Projection.Mul4(camera.Camera)
}
