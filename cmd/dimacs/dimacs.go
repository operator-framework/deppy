package dimacs

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// Dimacs constrains the variables and clauses that make up
// a CNF problem described in DIMACS format
// see: https://logic.pdmi.ras.ru/~basolver/dimacs.html
type Dimacs struct {
	variables []string
	clauses   []string
}

func (d *Dimacs) Variables() []string {
	return d.variables
}

func (d *Dimacs) Clauses() []string {
	return d.clauses
}

// NewDimacs creates a Dimacs struct with the values
// parsed from the DIMACS formatted stream afforted by dimacsReader
func NewDimacs(dimacsReader io.Reader) (*Dimacs, error) {
	reader := bufio.NewReader(dimacsReader)

	variableSet := map[string]struct{}{}
	numVariables := 0
	numClauses := 0
	var clauses []string

	commentLine := regexp.MustCompile(`^c\s*.*`)
	headerLine := regexp.MustCompile(`^p cnf\s+\d+\s+\d+\s*`)
	clauseLine := regexp.MustCompile(`^(-?\d+\s+)+0`)
	cleanInput := regexp.MustCompile(`\s\s+`)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("error reading dimacs data: %w", err)
		}
		line = strings.TrimSpace(line)
		line = strings.Trim(line, "\n")

		// ignore comments
		if commentLine.MatchString(line) {
			continue
		}

		// parse header
		if headerLine.MatchString(line) {
			line = cleanInput.ReplaceAllString(line, " ")
			problem := strings.Split(line, " ")
			if len(problem) != 4 {
				return nil, fmt.Errorf("invalid statement: (%s). Valid format is p cnf <variables> <clauses>", line)
			}
			numVariables, err = strconv.Atoi(problem[2])
			if err != nil {
				return nil, fmt.Errorf("invalid number (%s) in statement (%s)", problem[2], line)
			}
			numClauses, err = strconv.Atoi(problem[3])
			if err != nil {
				return nil, fmt.Errorf("invalid number (%s) in statement (%s)", problem[3], line)
			}
			clauses = make([]string, 0, numClauses)

			// parse next line
			continue
		}

		// collect clauses
		if clauseLine.MatchString(line) {
			if clauses == nil {
				return nil, fmt.Errorf("invalid dimacs format: missing header 'p cnf <variable> <clauses>'")
			}
			line = cleanInput.ReplaceAllString(line, " ")
			clause := strings.Split(line, " ")
			if clause[len(clause)-1] != "0" {
				return nil, fmt.Errorf("invalid clause (%s): does not end with 0", line)
			}
			clause = clause[:len(clause)-1]
			if err := validateClause(clause, numVariables); err != nil {
				return nil, fmt.Errorf("invalid clause (%s): %w", line, err)
			}

			// remember variables seen for final validation
			// to ensure number of variables declared in header
			// is the same as the number of variables used
			for _, lit := range clause {
				lit = strings.TrimPrefix(lit, "-")
				variableSet[lit] = struct{}{}
			}
			clauses = append(clauses, strings.Join(clause, " "))

			// parse next line
			continue
		}

		// error out if the instruction is invalid
		return nil, fmt.Errorf("invalid dimacs command: %s", line)
	}

	if numVariables == 0 || numClauses == 0 || clauses == nil {
		return nil, fmt.Errorf("invalid format: no variables or clauses found")
	}

	if len(clauses) != numClauses {
		return nil, fmt.Errorf("invalid format: number of clauses in header differ from the total number of clauses")
	}

	if len(variableSet) != numVariables {
		return nil, fmt.Errorf("invalid format: number of variables in header differ from the total number of unique variables found in clauses")
	}

	// create variables
	variables := make([]string, 0, numVariables)
	for i := 1; i <= numVariables; i++ {
		variables = append(variables, fmt.Sprint(i))
	}
	return &Dimacs{
		variables: variables,
		clauses:   clauses,
	}, nil
}

func validateClause(clause []string, numVariables int) error {
	for _, lit := range clause {
		litInt, err := strconv.Atoi(lit)
		if err != nil {
			return fmt.Errorf("%s is not a number", lit)
		}
		if litInt == 0 {
			return fmt.Errorf("0 is not a valid variable")
		}
		if litInt > numVariables || -1*litInt < -1*numVariables {
			return fmt.Errorf("%s is not a valid variable", lit)
		}
	}
	return nil
}
