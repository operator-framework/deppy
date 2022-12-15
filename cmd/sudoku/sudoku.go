package sudoku

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/operator-framework/deppy/pkg/input"
	"github.com/operator-framework/deppy/pkg/solver"
)

var _ input.EntitySource = &Sudoku{}
var _ input.VariableSource = &Sudoku{}

type Sudoku struct {
	*input.CacheEntitySource
}

func GetID(row int, col int, num int) solver.Identifier {
	n := num
	n += col * 9
	n += row * 81
	return solver.Identifier(fmt.Sprintf("%03d", n))
}

func NewSudoku() *Sudoku {
	var entities = make(map[solver.Identifier]input.Entity, 9*9*9)
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			for num := 0; num < 9; num++ {
				id := GetID(row, col, num)
				entities[id] = *input.NewEntity(id, map[string]string{
					"row": strconv.Itoa(row),
					"col": strconv.Itoa(col),
					"num": strconv.Itoa(num),
				})
			}
		}
	}
	return &Sudoku{
		CacheEntitySource: input.NewCacheQuerier(entities),
	}
}

func (s Sudoku) GetVariables(ctx context.Context, _ input.EntitySource) ([]solver.Variable, error) {
	// adapted from: https://github.com/go-air/gini/blob/871d828a26852598db2b88f436549634ba9533ff/sudoku_test.go#L10
	variables := make(map[solver.Identifier]*input.SimpleVariable, 0)
	inorder := make([]solver.Variable, 0)
	rand.Seed(time.Now().UnixNano())

	// create variables for all number in all positions of the board
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			for n := 0; n < 9; n++ {
				variable := input.NewSimpleVariable(solver.Identifier(GetID(row, col, n)))
				variables[variable.Identifier()] = variable
				inorder = append(inorder, variable)
			}
		}
	}

	// add a clause stating that every position on the board has a number
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			ids := make([]solver.Identifier, 9)
			for n := 0; n < 9; n++ {
				ids[n] = solver.Identifier(GetID(row, col, n))
			}
			// randomize order to create new sudoku boards every run
			rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })

			// create clause that the particular position has a number
			varID := solver.Identifier(fmt.Sprintf("%d-%d has a number", row, col))
			variable := input.NewSimpleVariable(varID, solver.Mandatory(), solver.Dependency(ids...))
			variables[varID] = variable
			inorder = append(inorder, variable)
		}
	}

	// every row has unique numbers
	for n := 0; n < 9; n++ {
		for row := 0; row < 9; row++ {
			for colA := 0; colA < 9; colA++ {
				idA := solver.Identifier(GetID(row, colA, n))
				variable := variables[idA]
				for colB := colA + 1; colB < 9; colB++ {
					variable.AddConstraint(solver.Conflict(solver.Identifier(GetID(row, colB, n))))
				}
			}
		}
	}

	// every column has unique numbers
	for n := 0; n < 9; n++ {
		for col := 0; col < 9; col++ {
			for rowA := 0; rowA < 9; rowA++ {
				idA := solver.Identifier(GetID(rowA, col, n))
				variable := variables[idA]
				for rowB := rowA + 1; rowB < 9; rowB++ {
					variable.AddConstraint(solver.Conflict(solver.Identifier(GetID(rowB, col, n))))
				}
			}
		}
	}

	// function adding constraints stating that every box on the board
	// rooted at x, y has unique numbers
	var box = func(x, y int) {
		// all offsets w.r.t. root x,y
		offs := []struct{ x, y int }{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {1, 2}, {2, 0}, {2, 1}, {2, 2}}
		// all numbers
		for n := 0; n < 9; n++ {
			for i, offA := range offs {
				idA := solver.Identifier(GetID(x+offA.x, y+offA.y, n))
				variable := variables[idA]
				for j := i + 1; j < len(offs); j++ {
					offB := offs[j]
					idB := solver.Identifier(GetID(x+offB.x, y+offB.y, n))
					variable.AddConstraint(solver.Conflict(idB))
				}
			}
		}
	}

	// every box has unique numbers
	for x := 0; x < 9; x += 3 {
		for y := 0; y < 9; y += 3 {
			box(x, y)
		}
	}

	return inorder, nil
}
