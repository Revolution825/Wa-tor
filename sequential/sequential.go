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

// Sequential Wa-Tor Simulation in Go

// package watorsequential implements a sequential Wa-Tor Simulation in Go
package watorsequential

import (
	"image/color"
	"log"
	"math/rand/v2"
	"time"

	"github.com/hajimehoshi/ebiten"
)

// scale defines the drawing scale for each cell
const scale = 1

// width and height define the size of the simulation grid
const width = 1800
const height = 1000

var blue color.Color = color.RGBA{69, 145, 196, 255}
var yellow color.Color = color.RGBA{255, 230, 120, 255}
var red color.Color = color.RGBA{255, 50, 50, 255}

// buffer is a temporary grid used for writing the updated state of the world
var buffer [width][height]square = [width][height]square{}
var count int = 0

// numShark is the number of sharks the simultaion starts with
var numShark int = 100

// numFish is the number of fish the simultaion starts with
var numFish int = 200000

// fishBreed is the number of simulation steps it takes for a fish to reproduce
var fishBreed int = 5

// sharkBreed is the number of simultion steps it takes for a shark to reproduce
var sharkBreed int = 10

// starve is the number of simulation steps it takes for a shark to starve
var starve int = 4

// energyGain is how much energy a shark gains after eating a fish
var energyGain = 2

// grid represents the current state of the world
var grid [width][height]square = [width][height]square{}

// square represents a cell in the simulation grid
//
// Fields:
//
//	typeId		0 = empty space, 1 = fish, 2 = shark
//	energy		shark energy
//	breedTimer	defines how long a fish or shark must live before breeding
type square struct {
	typeId     int
	energy     int
	breedTimer int
}

// chronon is used for tracking simulation steps
var chronon int = 0

// start is used for tracking elapsed time for measuring performance
var start = time.Now()

// Frame updates the simulation each Frame by calling the Update() function and the Display() function
//
// Parameters:
//
//	window — the Ebiten image buffer used for drawing.
//
// Returns:
//
//	error - if the Update step fails, nil otherwise
func Frame(window *ebiten.Image) error {
	count++
	chronon++
	var err error = nil

	if count == 1 {
		err = Update()
		count = 0
	}
	if !ebiten.IsDrawingSkipped() {
		Display(window)
	}
	if chronon == 1000 {
		var elapsed = time.Since(start)
		log.Printf("Elapsed time for 1000 chronons : %s", elapsed)
		chronon = 0
	}

	return err
}

// GatherFreeSquares takes in the coordinates of a particular square and returns a slice containing
// the coordinates of any empty squares to the north, south, east and west of the inputted coordinates
//
// Parameters:
//
//	x - x coordinate of current square
//	y - y coordinate of current square
//
// Returns:
//
//	[][2]int - containing the coordinates of all free squares, if there are no free squares returns empty slice
func GatherFreeSquares(x int, y int) [][2]int {
	freeSquares := [][2]int{}
	leftX := (x - 1 + width) % width
	rightX := (x + 1) % width
	upY := (y - 1 + height) % height
	downY := (y + 1) % height
	if grid[x][upY].typeId == 0 {
		freeSquares = append(freeSquares, [2]int{x, upY})
	}
	if grid[leftX][y].typeId == 0 {
		freeSquares = append(freeSquares, [2]int{leftX, y})
	}
	if grid[rightX][y].typeId == 0 {
		freeSquares = append(freeSquares, [2]int{rightX, y})
	}
	if grid[x][downY].typeId == 0 {
		freeSquares = append(freeSquares, [2]int{x, downY})
	}
	return freeSquares
}

// GatherFishSquares takes in the coordinates of a particular square and returns a slice containing
// the coordinates of any squares containing fish to the north, south, east and west of the inputted
// coordinates
//
// Parameters:
//
//	x - x coordinate of current square
//	y - y coordinate of current square
//
// Returns:
//
//	[][2]int - containing the coordinates of all fish squares, if there are no fish squares returns empty slice
func GatherFishSquares(x int, y int) [][2]int {
	fishSquares := [][2]int{}
	leftX := (x - 1 + width) % width
	rightX := (x + 1) % width
	upY := (y - 1 + height) % height
	downY := (y + 1) % height
	if grid[x][upY].typeId == 1 {
		fishSquares = append(fishSquares, [2]int{x, upY})
	}
	if grid[leftX][y].typeId == 1 {
		fishSquares = append(fishSquares, [2]int{leftX, y})
	}
	if grid[rightX][y].typeId == 1 {
		fishSquares = append(fishSquares, [2]int{rightX, y})
	}
	if grid[x][downY].typeId == 1 {
		fishSquares = append(fishSquares, [2]int{x, downY})
	}
	return fishSquares
}

// UpdateFish takes in the coordinates of a particular fish. It checks if this fish has been eaten yet
// in the buffer. If not, GatherFreeSquares is called. if there are free squares, one is picked at random and
// the buffer is checked to ensure that a different fish has not moved to the desired square already. If the
// empty square is still free in the buffer, the fish is written to that square. otherwise the fish stays put.
// UpdateFish also handles breeding by checking the moved fishes' breedtimer. if it is <=0 a new fish is
// placed in it's old place and both fishes' breedTimers are reset.
//
// Parameters:
//
//	x - x coordinate of current fish square
//	y - y coordinate of current fish square
//
// Returns:
//
//	nil
func UpdateFish(x int, y int) error {
	if buffer[x][y].typeId == 2 {
		return nil
	}
	freeSquares := GatherFreeSquares(x, y)
	newX, newY := x, y
	if len(freeSquares) > 0 {
		newPosition := rand.IntN(len(freeSquares))
		newX = freeSquares[newPosition][0]
		newY = freeSquares[newPosition][1]
		if buffer[newX][newY].typeId == 0 {
			buffer[newX][newY].typeId = 1
			buffer[newX][newY].breedTimer = grid[x][y].breedTimer - 1
		} else {
			buffer[x][y].typeId = grid[x][y].typeId
			buffer[x][y].breedTimer = grid[x][y].breedTimer - 1
		}
	} else {
		buffer[x][y].typeId = grid[x][y].typeId
		buffer[x][y].breedTimer = grid[x][y].breedTimer - 1
	}
	if buffer[newX][newY].breedTimer <= 0 {
		buffer[x][y].typeId = 1
		buffer[x][y].breedTimer = fishBreed
		buffer[newX][newY].breedTimer = fishBreed
	}
	return nil
}

// UpdateSharks takes in the coordinates of a particular shark. GatherFishSquares and GatherFreeSquares is called.
// if there are adjacent fish squares one is picked at random and the buffer is checked to ensure that the fish
// has not already been eaten by a different shark. If a shark cannot move to the desired fish square it attempts
// to move to a free square. If there are no free squares the shark stays put. A shark loses 1 energy per turn and
// gains a specified amount of energy upon eating a fish. If a sharks energy is <=0 it disappears. when a sharks
// breedtimer is <=0 a new shark is placed at its old position and both sharks' breedTimers reset.
//
// Parameters:
//
//	x - x coordinate of current shark square
//	y - y coordinate of current shark square
//
// Returns:
//
//	nil
func UpdateSharks(x int, y int) error {
	freeSquares := GatherFreeSquares(x, y)
	fishSquares := GatherFishSquares(x, y)
	newX, newY := x, y
	if len(fishSquares) > 0 {
		newPosition := rand.IntN(len(fishSquares))
		newX = fishSquares[newPosition][0]
		newY = fishSquares[newPosition][1]
		if buffer[newX][newY].typeId != 2 {
			buffer[newX][newY].typeId = 2
			buffer[newX][newY].energy = grid[x][y].energy - 1 + energyGain
			buffer[newX][newY].breedTimer = grid[x][y].breedTimer - 1
		} else if len(freeSquares) > 0 {
			newPosition := rand.IntN(len(freeSquares))
			newX = freeSquares[newPosition][0]
			newY = freeSquares[newPosition][1]
			buffer[newX][newY].typeId = 2
			buffer[newX][newY].energy = grid[x][y].energy - 1
			buffer[newX][newY].breedTimer = grid[x][y].breedTimer - 1
		} else {
			buffer[x][y].typeId = grid[x][y].typeId
			buffer[x][y].energy = grid[x][y].energy - 1
			buffer[x][y].breedTimer = grid[x][y].breedTimer - 1
			newX = x
			newY = y
		}
	} else if len(freeSquares) > 0 {
		newPosition := rand.IntN(len(freeSquares))
		newX = freeSquares[newPosition][0]
		newY = freeSquares[newPosition][1]
		buffer[newX][newY].typeId = 2
		buffer[newX][newY].energy = grid[x][y].energy - 1
		buffer[newX][newY].breedTimer = grid[x][y].breedTimer - 1
	} else {
		buffer[x][y].typeId = grid[x][y].typeId
		buffer[x][y].energy = grid[x][y].energy - 1
		buffer[x][y].breedTimer = grid[x][y].breedTimer - 1
	}
	if buffer[newX][newY].energy <= 0 {
		buffer[newX][newY].typeId = 0
		buffer[newX][newY].energy = 0
		return nil
	}
	if buffer[newX][newY].breedTimer <= 0 {
		buffer[x][y].typeId = 2
		buffer[x][y].energy = starve
		buffer[x][y].breedTimer = sharkBreed
		buffer[newX][newY].breedTimer = sharkBreed
	}
	return nil
}

// Update iterates through the grid (which represents the current state of the world), detects whether each cell
// contains a fish or a shark and calls the relevant function. When the main Update loop is complete it sets grid
// to be buffer (the now updated state of the world) and zeros the buffer.
//
// Returns:
//
//	nil
func Update() error {
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			if grid[x][y].typeId == 1 {
				UpdateFish(x, y)
			} else if grid[x][y].typeId == 2 {
				UpdateSharks(x, y)
			}
		}
	}

	grid = buffer

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			buffer[x][y] = square{}
		}
	}

	return nil
}

// Display draws the new grid after each Update loop
//
// Parameters:
//
//	window — the Ebiten image buffer used for drawing.
func Display(window *ebiten.Image) {
	window.Fill(blue)
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

// RunSequential initializes the grid and starts the sequential simulation loop
func RunSequential() {
	coords := [][2]int{}
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			coords = append(coords, [2]int{x, y})
		}
	}
	rand.Shuffle(len(coords), func(i, j int) {
		coords[i], coords[j] = coords[j], coords[i]
	})
	for i := 0; i < numFish; i++ {
		x := coords[i][0]
		y := coords[i][1]
		grid[x][y].typeId = 1
		grid[x][y].breedTimer = fishBreed
	}
	for i := numFish; i < numShark+numFish; i++ {
		x := coords[i][0]
		y := coords[i][1]
		grid[x][y].typeId = 2
		grid[x][y].breedTimer = sharkBreed
		grid[x][y].energy = starve
	}
	if err := ebiten.Run(Frame, width, height, 1, "Wa-tor Simulation (Sequential)"); err != nil {
		log.Fatal(err)
	}
}
