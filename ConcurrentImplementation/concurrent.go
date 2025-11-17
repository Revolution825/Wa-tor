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
const width = 800
const height = 800

var blue color.Color = color.RGBA{69, 145, 196, 255}
var yellow color.Color = color.RGBA{255, 230, 120, 255}
var red color.Color = color.RGBA{255, 50, 50, 255}
var buffer [width][height]square = [width][height]square{}
var count int = 0

var numShark int = 100000
var numFish int = 200000
var fishBreed int = 5
var sharkBreed int = 8
var starve int = 5
var energyGain = 4
var grid [width][height]square = [width][height]square{}
var threads int = 6

var tileWidth = width / threads
var tileLocks = make([]sync.Mutex, threads-1)

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

func updateFish(x int, y int, worker int) error {
	if buffer[x][y].typeId == 2 {
		return nil // skip if shark already moved here
	}
	freeSquares := gatherFreeSquares(x, y)
	newX, newY := x, y // Initialize newX and newY
	// If there are adjacent free squares, pick one at random and move there
	if len(freeSquares) > 0 {
		newPosition := rand.IntN(len(freeSquares))
		newX = freeSquares[newPosition][0]
		newY = freeSquares[newPosition][1]
		if buffer[newX][newY].typeId == 0 {
			safeWrite(newX, newY, square{
				typeId:     1,
				breedTimer: grid[x][y].breedTimer - 1,
			}, worker)
		} else {
			// No free adjacent squares, stay put
			safeWrite(x, y, square{
				typeId:     1,
				breedTimer: grid[x][y].breedTimer - 1,
			}, worker)
		}
	} else {
		// No free adjacent squares, stay put
		safeWrite(x, y, square{
			typeId:     grid[x][y].typeId,
			breedTimer: grid[x][y].breedTimer - 1,
		}, worker)
	}
	// Handle breeding
	if buffer[newX][newY].breedTimer <= 0 {
		safeWrite(newX, newY, square{
			typeId:     1,
			breedTimer: fishBreed,
		}, worker)
		safeWrite(x, y, square{
			typeId:     1,
			breedTimer: fishBreed,
		}, worker)
	}
	return nil
}

func updateSharks(x int, y int, worker int) error {
	freeSquares := gatherFreeSquares(x, y)
	fishSquares := gatherFishSquares(x, y)
	newX, newY := x, y // Initialize newX and newY
	if len(fishSquares) > 0 {
		// If there are adjacent fish squares, pick one at random and move there
		newPosition := rand.IntN(len(fishSquares))
		newX = fishSquares[newPosition][0]
		newY = fishSquares[newPosition][1]
		if buffer[newX][newY].typeId != 2 {
			safeWrite(newX, newY, square{
				typeId:     2,
				energy:     grid[x][y].energy - 1 + energyGain, // shark eats fish, gains energy
				breedTimer: grid[x][y].breedTimer - 1,
			}, worker)
		} else if len(freeSquares) > 0 {
			// if shark can't move to fish square because there's a shark there already in the buffer, try to move to free square
			newPosition := rand.IntN(len(freeSquares))
			newX = freeSquares[newPosition][0]
			newY = freeSquares[newPosition][1]
			safeWrite(newX, newY, square{
				typeId:     2,
				energy:     grid[x][y].energy - 1, // -1 energy for movement
				breedTimer: grid[x][y].breedTimer - 1,
			}, worker)
		} else {
			// No free adjacent squares, stay put
			safeWrite(x, y, square{
				typeId:     grid[x][y].typeId,
				energy:     grid[x][y].energy - 1,
				breedTimer: grid[x][y].breedTimer - 1,
			}, worker)
			newX = x
			newY = y
		}
	} else if len(freeSquares) > 0 {
		// if shark can't move to fish square because there's a shark there already in the buffer, try to move to free square
		newPosition := rand.IntN(len(freeSquares))
		newX = freeSquares[newPosition][0]
		newY = freeSquares[newPosition][1]
		safeWrite(newX, newY, square{
			typeId:     2,
			energy:     grid[x][y].energy - 1,
			breedTimer: grid[x][y].breedTimer - 1,
		}, worker)
	} else {
		// No free adjacent squares, stay put
		safeWrite(x, y, square{
			typeId:     grid[x][y].typeId,
			energy:     grid[x][y].energy - 1,
			breedTimer: grid[x][y].breedTimer - 1,
		}, worker)
	}
	// Handle starvation
	if buffer[newX][newY].energy <= 0 {
		safeWrite(newX, newY, square{
			typeId:     0,
			energy:     0,
			breedTimer: 0,
		}, worker)
		return nil
	}
	// Handle breeding
	if buffer[newX][newY].breedTimer <= 0 {
		safeWrite(x, y, square{
			typeId:     2,
			energy:     starve,
			breedTimer: sharkBreed,
		}, worker)
		safeWrite(newX, newY, square{
			typeId:     2,
			energy:     buffer[newX][newY].energy,
			breedTimer: sharkBreed,
		}, worker)
	}
	return nil
}

func whichTile(x int) int {
	return x / tileWidth
}

func safeWrite(x int, y int, square square, workerTile int) {
	targetTile := whichTile(x)

	// If writing to the current tile proceed as normal
	if targetTile == workerTile {
		buffer[x][y] = square
		return
	}

	// writing to the tile to the right
	if targetTile > workerTile {
		tileLocks[workerTile].Lock()
		buffer[x][y] = square
		tileLocks[workerTile].Unlock()
	}

	// writing to the tile to the left
	if targetTile < workerTile {
		tileLocks[targetTile].Lock()
		buffer[x][y] = square
		tileLocks[targetTile].Unlock()
	}
}

func update() error {
	var wg sync.WaitGroup
	for worker := 0; worker < threads; worker++ {
		// Split up tiles based on number of threads
		startX := worker * tileWidth
		endX := startX + tileWidth
		// Handle any remaining columns in the last worker
		if worker == threads-1 {
			endX = width
		}
		wg.Add(1)
		go concurrentUpdate(&wg, startX, endX, worker)
	}

	wg.Wait()

	grid = buffer

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			buffer[x][y] = square{}
		}
	}

	return nil
}

func concurrentUpdate(wg *sync.WaitGroup, startX int, endX int, worker int) {
	defer wg.Done()

	for x := startX; x < endX; x++ {
		for y := 0; y < height; y++ {
			if grid[x][y].typeId == 1 {
				updateFish(x, y, worker)
			} else if grid[x][y].typeId == 2 {
				updateSharks(x, y, worker)
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
