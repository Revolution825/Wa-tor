// Copyright (C) 2025  Diarmuid O'Neill

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Wa-Tor Simulation in Go

package main

import (
	"image/color"
	"log"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten"
)

const scale = 5
const width = 800
const height = 800

var blue color.Color = color.RGBA{69, 145, 196, 255}
var yellow color.Color = color.RGBA{255, 230, 120, 255}
var red color.Color = color.RGBA{255, 50, 50, 255}
var buffer [width][height]uint8 = [width][height]uint8{}
var count int = 0

var numShark int = 0
var numFish int = 0
var fishBreed int = 3
var sharkBreed int = 8
var starve int = 3
var grid [width][height]uint8 = [width][height]uint8{}
var threads int = 4

type shark struct {
	starveTimer int
	breedTimer  int
}

type fish struct {
	breedTimer int
}

func frame(window *ebiten.Image) error {
	count++
	var err error = nil

	if count == 1 {
		err = update()
		count = 0
	}
	if !ebiten.IsDrawingSkipped() {
		display(window)
	}

	return err
}

func update() error {
	var err error = nil

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			buffer[x][y] = 0

		}
	}
	return err
}

func display(window *ebiten.Image) {
	// Fills background blue
	window.Fill(blue)

	// Draws fish and sharks
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			for i := 0; i < scale; i++ {
				for j := 0; j < scale; j++ {
					if grid[x][y] == 1 {
						window.Set(x*scale+i, y*scale+j, yellow)
					} else if grid[x][y] == 2 {
						window.Set(x*scale+i, y*scale+j, red)
					}
				}
			}
		}
	}

}

func main() {
	// Initializes grid with random fish and sharks
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			// Float32() returns a random floating point number between 0.0 and 1.0
			if rand.Float32() < 0.333 {
				grid[x][y] = 1
			} else if rand.Float32() < 0.5 {
				grid[x][y] = 2
			} else {
				grid[x][y] = 0
			}
		}
	}

	// Starts simulation
	if err := ebiten.Run(frame, width, height, 1, "Wa-tor Simulation"); err != nil {
		log.Fatal(err)
	}
}
