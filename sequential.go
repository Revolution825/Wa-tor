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
const width = 400
const height = 400

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
	typeId     int // 0 = empty space, 1 = fish, 2 = shark
	energy     int
	breedTimer int
}

var chronon int = 1

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

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			if grid[x][y].typeId == 1 { // if fish
				freeSquares := [][2]int{}        // array of free square coordinates
				leftX := (x - 1 + width) % width // wrap around
				rightX := (x + 1) % width
				upY := (y - 1 + height) % height
				downY := (y + 1) % height
				if grid[x][upY].typeId == 0 && buffer[x][upY].typeId == 0 {
					freeSquares = append(freeSquares, [2]int{x, upY})
				}
				if grid[leftX][y].typeId == 0 && buffer[leftX][y].typeId == 0 {
					freeSquares = append(freeSquares, [2]int{leftX, y})
				}
				if grid[rightX][y].typeId == 0 && buffer[rightX][y].typeId == 0 {
					freeSquares = append(freeSquares, [2]int{rightX, y})
				}
				if grid[x][downY].typeId == 0 && buffer[x][downY].typeId == 0 {
					freeSquares = append(freeSquares, [2]int{x, downY})
				}
				if len(freeSquares) == 0 { // If there are no free squares, stay put
					buffer[x][y].typeId = grid[x][y].typeId
				} else { // If there are free squares, move to one at random
					newPosition := rand.IntN(len(freeSquares))
					newX := freeSquares[newPosition][0]
					newY := freeSquares[newPosition][1]
					if buffer[newX][newY].typeId == 0 {
						buffer[newX][newY].typeId = 1
						if grid[x][y].breedTimer <= 0 {
							buffer[x][y].typeId = 1
							buffer[x][y].breedTimer = fishBreed
							buffer[newX][newY].breedTimer = fishBreed
						} else {
							buffer[newX][newY].breedTimer = grid[x][y].breedTimer - 1
						}
					} else {
						buffer[x][y].typeId = grid[x][y].typeId
						buffer[x][y].breedTimer = grid[x][y].breedTimer - 1
					}
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

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			buffer[x][y].typeId = 0
		}
	}

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
			if rand.Float32() < 0.1 {
				grid[x][y].typeId = 1
				grid[x][y].breedTimer = fishBreed
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
