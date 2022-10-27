package sudoku

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/operator-framework/deppy/pkg/constraints"
	"github.com/operator-framework/deppy/pkg/entitysource"
	"github.com/operator-framework/deppy/pkg/entitysource/factory"
)

var _ entitysource.EntitySource = &Sudoku{}

var _ constraints.ConstraintGenerator = &Sudoku{}

type Sudoku struct {
	entitysource.EntityQuerier
	entitysource.EntityContentGetter
}

func GetID(row int, col int, num int) entitysource.EntityID {
	n := num
	n += col * 9
	n += row * 81
	return entitysource.EntityID(fmt.Sprintf("%03d", n))
}

func NewSudoku() *Sudoku {
	var entities = make(map[entitysource.EntityID]entitysource.Entity, 9*9*9)
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			for num := 0; num < 9; num++ {
				id := GetID(row, col, num)
				entities[id] = *entitysource.NewEntity(id, map[string]string{
					"row": strconv.Itoa(row),
					"col": strconv.Itoa(col),
					"num": strconv.Itoa(num),
				})
			}
		}
	}
	return &Sudoku{
		EntityQuerier:       factory.NewCacheQuerier(entities),
		EntityContentGetter: entitysource.NoContentSource(),
	}
}

func (s Sudoku) GetVariables(ctx context.Context, querier entitysource.EntityQuerier) ([]constraints.IVariable, error) {
	// adapted from: https://github.com/go-air/gini/blob/871d828a26852598db2b88f436549634ba9533ff/sudoku_test.go#L10
	variables := make(map[constraints.Identifier]*constraints.Variable, 0)
	inorder := make([]constraints.IVariable, 0)
	rand.Seed(time.Now().UnixNano())

	// create variables for all number in all positions of the board
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			for n := 0; n < 9; n++ {
				variable := constraints.NewVariable(constraints.Identifier(GetID(row, col, n)))
				variables[variable.Identifier()] = variable
				inorder = append(inorder, variable)
			}
		}
	}

	// add a clause stating that every position on the board has a number
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			ids := make([]constraints.Identifier, 9)
			for n := 0; n < 9; n++ {
				ids[n] = constraints.Identifier(GetID(row, col, n))
			}
			// randomize order to create new sudoku boards every run
			rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })

			// create clause that the particular position has a number
			varID := constraints.Identifier(fmt.Sprintf("%d-%d has a number", row, col))
			variable := constraints.NewVariable(varID, constraints.Mandatory(), constraints.Dependency(ids...))
			variables[varID] = variable
			inorder = append(inorder, variable)
		}
	}

	// every row has unique numbers
	for n := 0; n < 9; n++ {
		for row := 0; row < 9; row++ {
			for colA := 0; colA < 9; colA++ {
				idA := constraints.Identifier(GetID(row, colA, n))
				variable := variables[idA]
				for colB := colA + 1; colB < 9; colB++ {
					variable.AddConstraint(constraints.Conflict(constraints.Identifier(GetID(row, colB, n))))
				}
			}
		}
	}

	// every column has unique numbers
	for n := 0; n < 9; n++ {
		for col := 0; col < 9; col++ {
			for rowA := 0; rowA < 9; rowA++ {
				idA := constraints.Identifier(GetID(rowA, col, n))
				variable := variables[idA]
				for rowB := rowA + 1; rowB < 9; rowB++ {
					variable.AddConstraint(constraints.Conflict(constraints.Identifier(GetID(rowB, col, n))))
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
				idA := constraints.Identifier(GetID(x+offA.x, y+offA.y, n))
				variable := variables[idA]
				for j := i + 1; j < len(offs); j++ {
					offB := offs[j]
					idB := constraints.Identifier(GetID(x+offB.x, y+offB.y, n))
					variable.AddConstraint(constraints.Conflict(idB))
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
