package report

import (
	"errors"
	"fmt"
	"golang.org/x/tools/cover"
	"sort"
	"strings"
)

// Coverage summary for a file or module
type Summary struct {
	Name                                       string
	Blocks, Stmts, MissingBlocks, MissingStmts int
	BlockCoverage, StmtCoverage                float64
}

// Report of the coverage results
type Report struct {
	Total Summary // Global coverage
	Files []Summary // Coverage by file
}

// Generates a coverage report given the coverage profile file, and the following configurations:
// exclusions: packages to be excluded (if a package is excluded, all its subpackages are excluded as well)
// sortBy: the order in which the files will be sorted in the report (see sortResults)
// order: the direction of the the sorting
func GenerateReport(coverprofile string, root string, exclusions []string, sortBy, order string) (Report, error) {
	profiles, err := cover.ParseProfiles(coverprofile)
	if err != nil {
		return Report{}, fmt.Errorf("Invalid coverprofile: '%s'", err)
	}
	total := &accumulator{name: "Total"}
	files := make(map[string]*accumulator)
	for _, profile := range profiles {
		var fileName string
		if root == "" {
			fileName = profile.FileName
		} else {
			fileName = strings.Replace(profile.FileName, root+"/", "", -1)
		}
		skip := false
		for _, exclusion := range exclusions {
			if strings.HasPrefix(fileName, exclusion) {
				skip = true
			}
		}
		if skip {
			continue
		}
		fileCover, ok := files[fileName]
		if !ok {
			fileCover = &accumulator{name: fileName}
			files[fileName] = fileCover
		}
		for _, block := range profile.Blocks {
			total.add(block)
			fileCover.add(block)
		}
	}
	return makeReport(total, files, sortBy, order)
}

// Creates a Report struct from the coverage sumarization results
func makeReport(total *accumulator, files map[string]*accumulator, sortBy, order string) (Report, error) {
	fileReports := make([]Summary, 0, len(files))
	for _, fileCover := range files {
		fileReports = append(fileReports, fileCover.results())
	}
	if err := sortResults(fileReports, sortBy, order); err != nil {
		return Report{}, err
	}
	return Report{
		Total: total.results(),
		Files: fileReports}, nil
}

// Accumulates the coverage of a file and returns a summary
type accumulator struct {
	name                                       string
	blocks, stmts, coveredBlocks, coveredStmts int
}

// Accumulates a profile block
func (a *accumulator) add(block cover.ProfileBlock) {
	a.blocks++
	a.stmts += block.NumStmt
	if block.Count > 0 {
		a.coveredBlocks++
		a.coveredStmts += block.NumStmt
	}
}

func (a *accumulator) results() Summary {
	return Summary{
		Name:          a.name,
		Blocks:        a.blocks,
		Stmts:         a.stmts,
		MissingBlocks: a.blocks - a.coveredBlocks,
		MissingStmts:  a.stmts - a.coveredStmts,
		BlockCoverage: float64(a.coveredBlocks) / float64(a.blocks) * 100,
		StmtCoverage:  float64(a.coveredStmts) / float64(a.stmts) * 100}
}

// Sorts the individual coverage reports by a given column
// (block --block cover--, stmt, --stmt cover--, missing-blocks or missing-stmts)
// and a sorting direction (asc or desc)
func sortResults(reports []Summary, mode string, order string) error {
	var reverse bool
	var less func(i, j int) bool
	switch order {
	case "asc":
		reverse = false
	case "desc":
		reverse = true
	default:
		return errors.New("Order must be either asc or desc")
	}
	switch mode {
	case "filename":
		less = func(i, j int) bool {
			return reports[i].Name < reports[j].Name
		}
	case "block":
		less = func(i, j int) bool {
			return reports[i].BlockCoverage < reports[j].BlockCoverage
		}
	case "stmt":
		less = func(i, j int) bool {
			return reports[j].StmtCoverage < reports[j].StmtCoverage
		}
	case "missing-blocks":
		less = func(i, j int) bool {
			return reports[i].MissingBlocks < reports[j].MissingBlocks
		}
	case "missing-stmts":
		less = func(i, j int) bool {
			return reports[i].MissingStmts < reports[j].MissingStmts
		}
	default:
		return errors.New("Invalid sort colum, must be one of filename, block, stmt, missing-blocks or missing-stmts")
	}
	sort.Slice(reports, func(i, j int) bool {
		if reverse {
			return !less(i, j)
		} else {
			return less(i, j)
		}
	})
	return nil
}
