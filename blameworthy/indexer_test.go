package blameworthy

import (
	"fmt"
	"testing"
)

func TestStepping(t *testing.T) {
	var tests = []struct {
		inputCommits FileHistory
		expectedOutput string
	}{{
		FileHistory{},
		"[]",
	}, {
		FileHistory{
			FileCommit{"a1", []Hunk{
				Hunk{0,0,1,3},
			}},
		},
		"[[{3 1 a1}]]",
	}, {
		FileHistory{
			FileCommit{"a1", []Hunk{
				Hunk{0,0,1,3},
			}},
			FileCommit{"b2", []Hunk{
				Hunk{1,0,2,2},
				Hunk{2,0,5,2},
			}},
			FileCommit{"c3", []Hunk{
				Hunk{1,1,1,0},
				Hunk{4,2,3,1},
			}},
		},
		"[[{3 1 a1}]" +
			" [{1 1 a1} {2 2 b2} {1 2 a1} {2 5 b2} {1 3 a1}]" +
			" [{2 2 b2} {1 3 c3} {1 6 b2} {1 3 a1}]]",
	}, {
		FileHistory{
			FileCommit{"a1", []Hunk{
				Hunk{0,0,1,3},
			}},
			FileCommit{"b2", []Hunk{
				Hunk{1,1,0,0},  // remove 1st line
				Hunk{2,0,2,1},  // add new line 2
			}},
		},
		"[[{3 1 a1}] [{1 2 a1} {1 2 b2} {1 3 a1}]]",
	}, {
		FileHistory{
			FileCommit{"a1", []Hunk{
				Hunk{0,0,1,3},
			}},
			FileCommit{"b2", []Hunk{
				Hunk{1,3,0,0},
			}},
		},
		"[[{3 1 a1}] []]",
	}, {
		FileHistory{
			FileCommit{"a1", []Hunk{
				Hunk{0,0,1,3},
			}},
			FileCommit{"b2", []Hunk{
				Hunk{0,0,4,1},
			}},
		},
		"[[{3 1 a1}] [{3 1 a1} {1 4 b2}]]",
	}}
	for test_number, test := range tests {
		segments := BlameSegments{}
		out := []BlameSegments{}
		for _, commit := range test.inputCommits {
			segments = commit.step(segments)
			out = append(out, segments)
		}
		if (fmt.Sprint(out) != test.expectedOutput) {
			t.Error("Test", test_number + 1, "failed",
				"\n  Wanted", test.expectedOutput,
				"\n  Got   ", fmt.Sprint(out),
				"\n  From  ", test.inputCommits)
		}
	}
}

func TestAtMethod(t *testing.T) {
	var tests = []struct {
		inputCommits FileHistory
		expectedOutput string
	}{{
		FileHistory{
			FileCommit{"a1", []Hunk{
				Hunk{0,0,1,3},
			}},
		}, "" +
			"BLAME [{a1 1} {a1 2} {a1 3}]" +
			"FUTURE [{ 1} { 2} { 3}]",
	}, {
		FileHistory{
			FileCommit{"a1", []Hunk{
				Hunk{0,0,1,3},
			}},
			FileCommit{"b2", []Hunk{
				Hunk{1,0,2,2},
				Hunk{2,0,5,2},
			}},
			FileCommit{"c3", []Hunk{
				Hunk{1,1,1,0},
				Hunk{4,2,3,1},
			}},
		}, "" +
			"BLAME [{a1 1} {a1 2} {a1 3}]" +
			"FUTURE [{c3 1} {c3 4} { 5}]" +
			"BLAME [{a1 1} {b2 2} {b2 3} {a1 2} {b2 5} {b2 6} {a1 3}]" +
			"FUTURE [{c3 1} { 1} { 2} {c3 4} {c3 5} { 4} { 5}]" +
			"BLAME [{b2 2} {b2 3} {c3 3} {b2 6} {a1 3}]" +
			"FUTURE [{ 1} { 2} { 3} { 4} { 5}]",
	}, {
		FileHistory{
			FileCommit{"a1", []Hunk{
				Hunk{0,0,1,3},
			}},
			FileCommit{"b2", []Hunk{
				Hunk{1,1,0,0},  // remove 1st line
				Hunk{2,0,2,1},  // add new line 2
			}},
		}, "" +
			"BLAME [{a1 1} {a1 2} {a1 3}]" +
			"FUTURE [{b2 1} { 1} { 3}]" +
			"BLAME [{a1 2} {b2 2} {a1 3}]" +
			"FUTURE [{ 1} { 2} { 3}]",
	}, {
		FileHistory{
			FileCommit{"a1", []Hunk{
				Hunk{0,0,1,3},
			}},
			FileCommit{"b2", []Hunk{
				Hunk{1,3,0,0},
			}},
		}, "" +
			"BLAME [{a1 1} {a1 2} {a1 3}]" +
			"FUTURE [{b2 1} {b2 2} {b2 3}]" +
			"BLAME []" +
			"FUTURE []",
	}, {
		FileHistory{
			FileCommit{"a1", []Hunk{
				Hunk{0,0,1,3},
			}},
			FileCommit{"b2", []Hunk{
				Hunk{0,0,4,1},
			}},
		}, "" +
			"BLAME [{a1 1} {a1 2} {a1 3}]" +
			"FUTURE [{ 1} { 2} { 3}]" +
			"BLAME [{a1 1} {a1 2} {a1 3} {b2 4}]" +
			"FUTURE [{ 1} { 2} { 3} { 4}]",
	}}
	for test_number, test := range tests {
		out := ""
		for i := range(test.inputCommits) {
			blameVector, futureVector := test.inputCommits.FileBlame(i)
			out += fmt.Sprint("BLAME ", blameVector)
			out += fmt.Sprint("FUTURE ", futureVector)
		}
		if (fmt.Sprint(out) != test.expectedOutput) {
			t.Error("Test", test_number + 1, "failed",
				"\n  Wanted", test.expectedOutput,
				"\n  Got   ", out,
				"\n  From  ", test.inputCommits)
		}
	}
}
