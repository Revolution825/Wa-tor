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
var buffer [width][height]square = [width][height]square{}
var count int = 0

var numShark int = 0
var numFish int = 0
var fishBreed int = 3
var sharkBreed int = 8
var starve int = 3
var grid [width][height]square = [width][height]square{}
var threads int = 4

type square struct {
	typeId int // 0 = empty space, 1 = fish, 2 = shark
	energy int
}

var breedTimer int
var chronon int = 1

func frame(window *ebiten.Image) error {
	count++
	var err error = nil

	if count == 100 {
		err = update()
		count = 0
	}
	if !ebiten.IsDrawingSkipped() {
		display(window)
	}

	return err
}

func update() error {

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			buffer[x][y].typeId = 0
		}
	}

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			if grid[x][y].typeId == 1 { // if fish
				freeSquares := [][2]int{} // array of free square coordinates
				if y > 0 && grid[x][y-1].typeId == 0 {
					freeSquares = append(freeSquares, [2]int{x, y - 1})
				}
				if x > 0 && grid[x-1][y].typeId == 0 {
					freeSquares = append(freeSquares, [2]int{x - 1, y})
				}
				if x < width-1 && grid[x+1][y].typeId == 0 {
					freeSquares = append(freeSquares, [2]int{x + 1, y})
				}
				if y < height-1 && grid[x][y+1].typeId == 0 {
					freeSquares = append(freeSquares, [2]int{x, y + 1})
				}
				if len(freeSquares) == 0 { // If there are no free squares, stay put
					buffer[x][y].typeId = grid[x][y].typeId
				} else { // If there are free squares, move to one at random
					newPosition := rand.IntN(len(freeSquares))
					buffer[freeSquares[newPosition][0]][freeSquares[newPosition][1]].typeId = 1
				}

			} else if grid[x][y].typeId == 2 { // if shark
				//TODO: implement shark logic
			}
		}
	}

	chronon++
	temp := buffer
	buffer = grid
	grid = temp
	return nil
}

func display(window *ebiten.Image) {
	// Fills background blue
	window.Fill(blue)

	// Draws fish and sharks
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			for i := 0; i < scale; i++ {
				for j := 0; j < scale; j++ {
					if grid[x][y].typeId == 1 {
						window.Set(x*scale+i, y*scale+j, yellow)
					} else if grid[x][y].typeId == 2 {
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
				grid[x][y].typeId = 1
			} else if rand.Float32() < 0.5 {
				grid[x][y].typeId = 2
			} else {
				grid[x][y].typeId = 0
			}
		}
	}

	// Starts simulation
	if err := ebiten.Run(frame, width, height, 1, "Wa-tor Simulation"); err != nil {
		log.Fatal(err)
	}
}
