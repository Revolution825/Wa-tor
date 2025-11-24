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

// package watorconcurrent implements a concurrent Wa-Tor Simulation in Go
package watorconcurrent

import (
	"image/color"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten"
)

// scale defines the drawing scale for each cell.
const scale = 1

// width and height define the size of the simulation grid.
const width = 1800
const height = 1000

var blue color.Color = color.RGBA{69, 145, 196, 255}
var yellow color.Color = color.RGBA{255, 230, 120, 255}
var red color.Color = color.RGBA{255, 50, 50, 255}

// buffer is a temporary grid used for writing the updated state of the world.
var buffer [width][height]square = [width][height]square{}
var count int = 0

// numShark is the number of sharks the simultaion starts with.
var numShark int = 100

// numFish is the number of fish the simultaion starts with.
var numFish int = 200000

// fishBreed is the number of simulation steps it takes for a fish to reproduce.
var fishBreed int = 5

// sharkBreed is the number of simultion steps it takes for a shark to reproduce.
var sharkBreed int = 10

// starve is the number of simulation steps it takes for a shark to starve.
var starve int = 4

// energyGain is how much energy a shark gains after eating a fish.
var energyGain = 2

// grid represents the current state of the world
var grid [width][height]square = [width][height]square{}

// threads represents the number of threads the simulation runs on when running concurrently.
var threads int = 8

// tileWidth represents the width of a tile ie. how many columns each tile contains.
var tileWidth = width / threads

// tileLocks is a slice of mutexes for each thread.
var tileLocks = make([]sync.Mutex, threads)

// starts is a slice of ints representing the x values of where each tile starts.
var starts = GetTileStarts(threads)

// square represents a cell in the simulation grid.
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

// chronon is used for tracking simulation steps.
var chronon int = 0

// start is used for tracking elapsed time for measuring performance.
var start = time.Now()

// Frame updates the simulation each Frame by calling the Update() function and the Display() function.
//
// Parameters:
//
//	window — the Ebiten image buffer used for drawing.
//
// Returns:
//
//	error - if the Update step fails. nil otherwise.
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
// the coordinates of any empty squares to the north, south, east and west of the inputted coordinates.
//
// Parameters:
//
//	 x - x coordinate of current square.
//		Y - y coordinate of current square.
//
// Returns:
//
//	[][2]int - containing the coordinates of all free squares. if there are no free squares returns empty slice.
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
// coordinates.
//
// Parameters:
//
//	x - x coordinate of current square
//	y - y coordinate of current square
//
// Returns:
//
//	[][2]int - containing the coordinates of all fish squares. if there are no fish squares returns empty slice.
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

// UpdateFish takes in the coordinates of a particular fish. It checks if this fish is still in the buffer.
// If not, GatherFreeSquares is called. if there are free squares, one is picked at random and SafeWrite
// attempts to write to the new coordinates. If SafeWrite fails or there are no adjacent cells free the fish
// stays put. UpdateFish also handles breeding by checking the moved fishes' breedtimer. if it is <=0 a new fish is
// placed in it's old place and both fishes' breedTimers are reset.
//
// Parameters:
//
//	x - x coordinate of current fish square.
//	y - y coordinate of current fish square.
//	worker int - current tile/thread we are working on.
//	starts []int - represents x values of where each tile starts.
//
// Returns:
//
//	nil
func UpdateFish(x int, y int, worker int, starts []int) error {
	currentSquare := grid[x][y]
	if currentSquare.typeId != 1 {
		return nil
	}
	freeSquares := GatherFreeSquares(x, y)
	newX, newY := x, y
	moved := false
	if len(freeSquares) > 0 {
		newPosition := rand.IntN(len(freeSquares))
		newX = freeSquares[newPosition][0]
		newY = freeSquares[newPosition][1]
		moved = SafeWrite(newX, newY, square{
			typeId:     1,
			breedTimer: currentSquare.breedTimer - 1,
		}, worker, starts)
		if !moved {
			SafeWrite(x, y, square{
				typeId:     1,
				breedTimer: currentSquare.breedTimer - 1,
			}, worker, starts)
			newX = x
			newY = y
		}
	} else {
		SafeWrite(x, y, square{
			typeId:     1,
			breedTimer: currentSquare.breedTimer - 1,
		}, worker, starts)
	}
	if currentSquare.breedTimer <= 0 {
		SafeWrite(x, y, square{
			typeId:     1,
			breedTimer: fishBreed,
		}, worker, starts)
		SafeWrite(newX, newY, square{
			typeId:     1,
			breedTimer: fishBreed,
		}, worker, starts)
	}
	return nil
}

// UpdateSharks takes in the coordinates of a particular shark. GatherFishSquares and GatherFreeSquares is called.
// if there are adjacent fish squares one is picked at random and SafeWrite attempts to write to the new coordinates.
// If SafeWrite fails the shark will stay put. If there are no fish squares but there are free squares one is picked
// at random and SafeWrite attempts to write to the new coordinates. If SafeWrite fails the shark will stay put.
// A shark loses 1 energy per turn and gains a specified amount of energy upon eating a fish. If a sharks energy is
// <=0 it disappears. when a sharks breedtimer is <=0 a new shark is placed at its old position and both sharks'
// breedTimers reset.
//
// Parameters:
//
//	x - x coordinate of current fish square.
//	y - y coordinate of current fish square.
//	worker int - current tile/thread we are working on.
//	starts []int - represents x values of where each tile starts.
//
// Returns:
//
//	nil
func UpdateSharks(x int, y int, worker int, starts []int) error {
	currentSquare := grid[x][y]
	if currentSquare.typeId != 2 {
		return nil
	}
	freeSquares := GatherFreeSquares(x, y)
	fishSquares := GatherFishSquares(x, y)
	newX, newY := x, y
	energyAfterMove := currentSquare.energy - 1
	moved := false
	if len(fishSquares) > 0 {
		newPosition := rand.IntN(len(fishSquares))
		newX = fishSquares[newPosition][0]
		newY = fishSquares[newPosition][1]
		moved = SafeWrite(newX, newY, square{
			typeId:     2,
			energy:     energyAfterMove + energyGain,
			breedTimer: currentSquare.breedTimer - 1,
		}, worker, starts)
		if moved {
			energyAfterMove += energyGain
		} else {
			SafeWrite(x, y, square{
				typeId:     2,
				energy:     energyAfterMove,
				breedTimer: currentSquare.breedTimer - 1,
			}, worker, starts)
		}
	} else if len(freeSquares) > 0 {
		newPosition := rand.IntN(len(freeSquares))
		newX = freeSquares[newPosition][0]
		newY = freeSquares[newPosition][1]
		moved = SafeWrite(newX, newY, square{
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
		SafeWrite(newX, newY, square{
			typeId:     2,
			energy:     energyAfterMove,
			breedTimer: currentSquare.breedTimer - 1,
		}, worker, starts)
	}
	if energyAfterMove <= 0 {
		SafeWrite(newX, newY, square{
			typeId:     0,
			energy:     0,
			breedTimer: 0,
		}, worker, starts)
		return nil
	}
	if currentSquare.breedTimer <= 0 {
		SafeWrite(newX, newY, square{
			typeId:     2,
			energy:     energyAfterMove,
			breedTimer: sharkBreed,
		}, worker, starts)
		SafeWrite(x, y, square{
			typeId:     2,
			energy:     starve,
			breedTimer: sharkBreed,
		}, worker, starts)
	}
	return nil
}

// TileOfX returns which tile the current inputted x value exists on.
//
// Parameters:
//
//	x - current x value.
//	starts - slice of ints representing where each tile starts.
//
// Returns:
//
//	int - coresponding with the tile the current x value exists on.
func TileOfX(x int, starts []int) int {
	for i := 0; i < len(starts)-1; i++ {
		if x >= starts[i] && x < starts[i+1] {
			return i
		}
	}
	return len(starts) - 1
}

// GetTileStarts returns a slice of ints representing the x values of where each tile starts.
//
// Parameters:
//
//	threads int - number of threads.
//
// Returns:
//
//	[]int - slice of ints representing the x values of where each tile starts.
func GetTileStarts(threads int) []int {
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

// SafeWrite checks whether or not the coordinates that are being written to are in the current tile (no lock will be needed)
// or if it is in a different tile (a lock will be needed). It also checks the buffer to ensure only legal writes are allowed.
//
// Parameters:
//
//	x int - x coordinate of new square.
//	y int - y coordinate of new square.
//	square square - details of the square we want to SafeWrite.
//	workerTile int - current tile/thread we are working on.
//	starts []int - slice representing x values of where each tile starts.
//
// Returns:
//
//	bool - returns whether or not the SafeWrite was successful.
func SafeWrite(x int, y int, square square, workerTile int, starts []int) bool {
	targetTile := TileOfX(x, starts)
	if targetTile == workerTile {
		existing := buffer[x][y]
		if existing.typeId == 0 {
			buffer[x][y] = square
			return true
		}
		if existing.typeId == 1 && square.typeId == 2 {
			buffer[x][y] = square
			return true
		}
		if existing.typeId == 2 && square.typeId == 0 {
			buffer[x][y] = square
			return true
		}
		return false
	}

	tileLocks[targetTile].Lock()
	defer tileLocks[targetTile].Unlock()
	existing := buffer[x][y]
	if existing.typeId == 0 {
		buffer[x][y] = square
		return true
	}
	if existing.typeId == 1 && square.typeId == 2 {
		buffer[x][y] = square
		return true
	}
	if existing.typeId == 2 && square.typeId == 0 {
		buffer[x][y] = square
		return true
	}
	return false
}

// Update splits tiles up based on number of threads, calls ConcurrentUpdate, waits for all routines to finish,
// sets grid to be buffer, and zeros the buffer each Frame.
//
// Returns:
//
//	nil
func Update() error {
	var wg sync.WaitGroup
	for worker := 0; worker < threads; worker++ {
		startX := starts[worker]
		endX := starts[worker+1]
		if endX > width {
			endX = width
		}
		wg.Add(1)
		go ConcurrentUpdate(&wg, startX, endX, worker, starts)
	}

	wg.Wait()

	grid = buffer

	buffer = [width][height]square{}

	return nil
}

// ConcurrentUpdate iterates through a specified tile in the grid and detects whether each cell
// contains a fish or a shark and calls the relevant function.
//
// Parameters:
//
//	wg *sync.WaitGroup - waitgroup for synchronization.
//	startX int - tile start.
//	endX int - tile end.
//	worker int - current tile number.
//	starts []int - slice representing x values of where each tile starts.
func ConcurrentUpdate(wg *sync.WaitGroup, startX int, endX int, worker int, starts []int) {
	defer wg.Done()

	for x := startX; x < endX; x++ {
		for y := 0; y < height; y++ {
			if grid[x][y].typeId == 1 {
				UpdateFish(x, y, worker, starts)
			} else if grid[x][y].typeId == 2 {
				UpdateSharks(x, y, worker, starts)
			}
		}
	}
}

// Display draws the new grid after each Update loop.
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

// RunConcurrent initializes the grid and starts the concurrent simulation loop.
func RunConcurrent() {
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
	if err := ebiten.Run(Frame, width, height, 1, "Wa-tor Simulation (Concurrent)"); err != nil {
		log.Fatal(err)
	}
}
