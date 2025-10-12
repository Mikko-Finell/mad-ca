package render

import (
	"errors"

	"github.com/hajimehoshi/ebiten/v2"
)

// GridShaderSource returns nil until a custom shader is wired in.
func GridShaderSource() []byte { return nil }

// NewGridShader currently returns an error, serving as a placeholder hook.
func NewGridShader() (*ebiten.Shader, error) {
	return nil, errors.New("grid shader not implemented")
}
