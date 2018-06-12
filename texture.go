package main

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"os"

	"github.com/go-gl/gl/v4.1-core/gl"
)

type Texture struct {
	Path string
	RGBA *image.RGBA
	ID   uint32
}

func LoadTexture(path string) (*Texture, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("texture %q not found on disk: %v", path, err)
	}

	m, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("unable to decode texture %q: %v", path, err)
	}

	rgba := image.NewRGBA(m.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return nil, fmt.Errorf("unsupported stride")
	}
	draw.Draw(rgba, rgba.Bounds(), m, image.Point{0, 0}, draw.Src)

	texture := &Texture{}
	texture.Path = path
	texture.RGBA = rgba

	texture.upload()

	return texture, nil
}

func (texture *Texture) upload() {
	if texture.ID != 0 {
		texture.delete()
	}

	gl.GenTextures(1, &texture.ID)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture.ID)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(texture.RGBA.Rect.Dx()),
		int32(texture.RGBA.Rect.Dy()),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(texture.RGBA.Pix))
}

func (texture *Texture) delete() {
	gl.DeleteTextures(1, &texture.ID)
	texture.ID = 0
}

func (texture *Texture) Destroy() {
	texture.delete()
	texture.RGBA = nil
	texture.Path = ""
}
