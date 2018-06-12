package main

var cubeVertices = []float32{
	//  X, Y, Z, U, V
	-1.0, -1.0, -1.0, 0.0, 0.0,
	1.0, -1.0, -1.0, 1.0, 0.0,
	-1.0, -1.0, 1.0, 0.0, 1.0,
	1.0, -1.0, 1.0, 1.0, 1.0,
	-1.0, 1.0, -1.0, 0.0, 0.0,
	-1.0, 1.0, 1.0, 0.0, 1.0,
	1.0, 1.0, -1.0, 1.0, 0.0,
	1.0, 1.0, 1.0, 1.0, 1.0,
	-1.0, -1.0, 1.0, 1.0, 0.0,
	1.0, -1.0, 1.0, 0.0, 0.0,
	-1.0, 1.0, 1.0, 1.0, 1.0,
	1.0, 1.0, 1.0, 0.0, 1.0,
	-1.0, 1.0, -1.0, 0.0, 1.0,
	1.0, 1.0, -1.0, 1.0, 1.0,
	-1.0, 1.0, -1.0, 1.0, 0.0,
	1.0, 1.0, -1.0, 0.0, 0.0,
}

var cubeIndices = []int8{
	0, 1, 2, 1, 3, 2,
	4, 5, 6, 6, 5, 7,
	8, 9, 10, 9, 11, 10,
	0, 12, 1, 1, 12, 13,
	2, 14, 0, 2, 10, 14,
	3, 1, 15, 3, 15, 11,
}
