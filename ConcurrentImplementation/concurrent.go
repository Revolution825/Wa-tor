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

// Concurrent Wa-Tor Simulation in Go

package main

import (
	"image/color"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten"
)

const scale = 1
const width = 1800
const height = 1000

var blue color.Color = color.RGBA{69, 145, 196, 255}
var yellow color.Color = color.RGBA{255, 230, 120, 255}
var red color.Color = color.RGBA{255, 50, 50, 255}
var buffer [width][height]square = [width][height]square{}
var count int = 0

var numShark int = 100
var numFish int = 200000
var fishBreed int = 5
var sharkBreed int = 10
var starve int = 4
var energyGain = 2
var grid [width][height]square = [width][height]square{}
var threads int = 8

var tileWidth = width / threads
var tileLocks = make([]sync.Mutex, threads)
var starts = getTileStarts(threads)

type square struct {
	typeId     int // 0 = empty space, 1 = fish, 2 = shark
	energy     int
	breedTimer int
}

var chronon int = 0
var start = time.Now()

func frame(window *ebiten.Image) error {
	count++
	chronon++
	var err error = nil

	if count == 1 {
		err = update()
		count = 0
	}
	if !ebiten.IsDrawingSkipped() {
		display(window)
	}

	if chronon == 1000 {
		var elapsed = time.Since(start)
		log.Printf("Elapsed time for 1000 chronons : %s", elapsed)
		chronon = 0
	}

	return err
}

func gatherFreeSquares(x int, y int) [][2]int {
	freeSquares := [][2]int{}
	leftX := (x - 1 + width) % width // wrap around
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

func gatherFishSquares(x int, y int) [][2]int {
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

func updateFish(x int, y int, worker int, starts []int) error {
	// if a fish is not at the current position, return early
	currentSquare := grid[x][y]
	if currentSquare.typeId != 1 {
		return nil
	}
	freeSquares := gatherFreeSquares(x, y)
	newX, newY := x, y // Initialize newX and newY
	moved := false

	// If there are adjacent free squares, pick one at random and move there
	if len(freeSquares) > 0 {
		newPosition := rand.IntN(len(freeSquares))
		newX = freeSquares[newPosition][0]
		newY = freeSquares[newPosition][1]

		moved = safeWrite(newX, newY, square{
			typeId:     1,
			breedTimer: currentSquare.breedTimer - 1,
		}, worker, starts)

		if !moved {
			// Failed to move, stay put
			safeWrite(x, y, square{
				typeId:     1,
				breedTimer: currentSquare.breedTimer - 1,
			}, worker, starts)
			newX = x
			newY = y
		}
	} else {
		// No free adjacent squares, stay put
		safeWrite(x, y, square{
			typeId:     1,
			breedTimer: currentSquare.breedTimer - 1,
		}, worker, starts)
	}
	// Handle breeding
	if currentSquare.breedTimer <= 0 {
		// Place new fish at original position
		safeWrite(x, y, square{
			typeId:     1,
			breedTimer: fishBreed,
		}, worker, starts)

		// Reset breed timer for moved fish
		safeWrite(newX, newY, square{
			typeId:     1,
			breedTimer: fishBreed,
		}, worker, starts)
	}
	return nil
}

func updateSharks(x int, y int, worker int, starts []int) error {
	currentSquare := grid[x][y]
	// if a shark is not at the current position, return early
	if currentSquare.typeId != 2 {
		return nil
	}
	freeSquares := gatherFreeSquares(x, y)
	fishSquares := gatherFishSquares(x, y)

	newX, newY := x, y // Initialize newX and newY
	energyAfterMove := currentSquare.energy - 1
	moved := false

	if len(fishSquares) > 0 {
		// If there are adjacent fish squares, pick one at random and move there
		newPosition := rand.IntN(len(fishSquares))
		newX = fishSquares[newPosition][0]
		newY = fishSquares[newPosition][1]

		moved = safeWrite(newX, newY, square{
			typeId:     2,
			energy:     energyAfterMove + energyGain, // shark eats fish, gains energy
			breedTimer: currentSquare.breedTimer - 1,
		}, worker, starts)

		if moved {
			energyAfterMove += energyGain
		} else {
			// Failed to move, stay put
			safeWrite(x, y, square{
				typeId:     2,
				energy:     energyAfterMove,
				breedTimer: currentSquare.breedTimer - 1,
			}, worker, starts)
		}

	} else if len(freeSquares) > 0 {
		// if shark can't move to fish square because there's a shark there already in the buffer, try to move to free square
		newPosition := rand.IntN(len(freeSquares))
		newX = freeSquares[newPosition][0]
		newY = freeSquares[newPosition][1]

		moved = safeWrite(newX, newY, square{
			typeId:     2,
			energy:     energyAfterMove,
			breedTimer: currentSquare.breedTimer - 1,
		}, worker, starts)

		if !moved {
			newX = x
			newY = y
		}
	} else {
		newX = x
		newY = y
	}

	if !moved {
		// No free adjacent squares, stay put
		safeWrite(newX, newY, square{
			typeId:     2,
			energy:     energyAfterMove,
			breedTimer: currentSquare.breedTimer - 1,
		}, worker, starts)
	}

	// Handle starvation
	if energyAfterMove <= 0 {
		safeWrite(newX, newY, square{
			typeId:     0,
			energy:     0,
			breedTimer: 0,
		}, worker, starts)
		return nil
	}
	// Handle breeding
	if currentSquare.breedTimer <= 0 {
		// Reset breed timer for moved shark
		safeWrite(newX, newY, square{
			typeId:     2,
			energy:     energyAfterMove,
			breedTimer: sharkBreed,
		}, worker, starts)

		// Place new shark at original position
		safeWrite(x, y, square{
			typeId:     2,
			energy:     starve,
			breedTimer: sharkBreed,
		}, worker, starts)
	}
	return nil
}

func tileOfX(x int, starts []int) int {
	// Returns which tile x is in
	for i := 0; i < len(starts)-1; i++ {
		if x >= starts[i] && x < starts[i+1] {
			return i
		}
	}
	return len(starts) - 1
}

func getTileStarts(threads int) []int {
	remainingWidth := width % threads
	starts := make([]int, threads+1)
	position := 0
	for i := 0; i < threads; i++ {
		starts[i] = position
		add := tileWidth
		if i == threads-1 {
			add += remainingWidth
		}
		position += add
	}
	starts[threads] = position
	return starts
}

func safeWrite(x int, y int, square square, workerTile int, starts []int) bool {
	targetTile := tileOfX(x, starts)

	// Same tile, no locking needed
	if targetTile == workerTile {
		existing := buffer[x][y]

		// Empty square, safe to write
		if existing.typeId == 0 {
			buffer[x][y] = square
			return true
		}

		// Shark can overwrite fish
		if existing.typeId == 1 && square.typeId == 2 {
			buffer[x][y] = square
			return true
		}

		// shark can overwrite empty, allows starvation
		if existing.typeId == 2 && square.typeId == 0 {
			buffer[x][y] = square
			return true
		}

		// already occupied
		return false
	}

	tileLocks[targetTile].Lock()
	defer tileLocks[targetTile].Unlock()

	existing := buffer[x][y]

	// Empty square, safe to write
	if existing.typeId == 0 {
		buffer[x][y] = square
		return true
	}

	// Shark can overwrite fish
	if existing.typeId == 1 && square.typeId == 2 {
		buffer[x][y] = square
		return true
	}

	// shark can overwrite empty, allows starvation
	if existing.typeId == 2 && square.typeId == 0 {
		buffer[x][y] = square
		return true
	}

	return false
}

func update() error {
	var wg sync.WaitGroup
	for worker := 0; worker < threads; worker++ {
		// Split up tiles based on number of threads
		startX := starts[worker]
		endX := starts[worker+1]
		if endX > width {
			endX = width
		}
		wg.Add(1)
		go concurrentUpdate(&wg, startX, endX, worker, starts)
	}

	wg.Wait()

	grid = buffer

	buffer = [width][height]square{}

	return nil
}

func concurrentUpdate(wg *sync.WaitGroup, startX int, endX int, worker int, starts []int) {
	defer wg.Done()

	for x := startX; x < endX; x++ {
		for y := 0; y < height; y++ {
			if grid[x][y].typeId == 1 {
				updateFish(x, y, worker, starts)
			} else if grid[x][y].typeId == 2 {
				updateSharks(x, y, worker, starts)
			}
		}
	}
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

	// Starts simulation
	if err := ebiten.Run(frame, width, height, 1, "Wa-tor Simulation (Concurrent)"); err != nil {
		log.Fatal(err)
	}
}
