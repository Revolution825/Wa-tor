# Wa-tor

Author: Diarmuid O'Neill (C00282898@setu.ie) <br />
Date: 26/11/2025 <br />
Brief Description: <br />
This repository demonstrates sequential and concurrent Wa-Tor simulations in GO language (see description below). The goal of this project is to demonstrate tiling, write thread-safe concurrent code, and measure execution speedup as the number of threads increases (see excel for results). This project was created for the final year concurrent development module of the Software Development course at SETU Carlow. To run the simulations, navigate to the main_sequential or main_concurrent directories (depending on which version you wish to run) and run the programs from there.

GitHub Link: https://github.com/Revolution825/Wa-tor.git

# Description

This is an excerpt from Wikipedia. For the full description of Wa-Tor see: https://en.wikipedia.org/wiki/Wa-Tor. <br />
Time passes in discrete jumps, which I shall call chronons. During each chronon a fish or shark may
move north, east, south or west to an adjacent point, provided the point is not already occupied by a
member of its own species. A random-number generator makes the actual choice. For a fish the
choice is simple: select one unoccupied adjacent point at random and move there. If all four adjacent
points are occupied, the fish does not move. Since hunting for fish takes priority over mere
movement, the rules for a shark are more complicated: from the adjacent points occupied by fish,
select one at random, move there and devour the fish. If no fish are in the neighborhood, the shark
moves just as a fish does, avoiding its fellow sharks.
### Fish
* At each chronon, a fish moves randomly to one of the adjacent unoccupied squares. If there are no
free squares, no movement takes place.
* Once a fish has survived a certain number of chronons it may reproduce. This is done as it moves
to a neighbouring square, leaving behind a new fish in its old position. Its reproduction time is
also reset to zero.
### Sharks
* At each chronon, a shark moves randomly to an adjacent square occupied by a fish. If there is
none, the shark moves to a random adjacent unoccupied square. If there are no free squares, no
movement takes place.
* At each chronon, each shark is deprived of a unit of energy
* Upon reaching zero energy, a shark dies.
* If a shark moves to a square occupied by a fish, it eats the fish and earns a certain amount of
energy.
* Once a shark has survived a certain number of chronons it may reproduce in exactly the same
way as the fish.
