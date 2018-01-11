package blameworthy

import (
	"reflect"
	"strings"
	"testing"
)

type HalfBlameResult struct {
	PreviousCommit string
	Vector BlameVector
}

type ExpectedHalfBlameResults map[string]HalfBlameResult

func TestHalfIndexing(t *testing.T) {
	var tests = []struct {
		input_commits CommitHistory
		expected_output ExpectedHalfBlameResults
	}{{
		CommitHistory{},
		ExpectedHalfBlameResults{},
	}, {
		CommitHistory{
			Commit{"a1", []FileHunks{
				FileHunks{"README", []Hunk{
					Hunk{0,0,1,3},
				}},
			}},
		},
		ExpectedHalfBlameResults{
			"a1:README": {"",
				BlameVector{"a1","a1","a1"}},
		},
	}, {
		CommitHistory{
			Commit{"a1", []FileHunks{
				FileHunks{"README", []Hunk{
					Hunk{0,0,1,3},
				}},
			}},
			Commit{"b2", []FileHunks{
				FileHunks{"README", []Hunk{
					Hunk{1,0,2,2},
					Hunk{2,0,5,2},
				}},
			}},
			Commit{"c3", []FileHunks{
				FileHunks{"README", []Hunk{
					Hunk{1,1,1,0},
					Hunk{4,2,3,1},
				}},
			}},
		},
		ExpectedHalfBlameResults{
			"a1:README": {"",
				BlameVector{"a1","a1","a1"}},
			"b2:README": {"a1",
				BlameVector{"a1","b2","b2","a1","b2","b2","a1"}},
			"c3:README": {"b2",
				BlameVector{"b2","b2","c3","b2","a1"}},
		},
	}, {
		CommitHistory{
			Commit{"a1", []FileHunks{
				FileHunks{"README", []Hunk{
					Hunk{0,0,1,3},
				}},
			}},
			Commit{"b2", []FileHunks{
				FileHunks{"README", []Hunk{
					Hunk{1,1,0,0},  // remove 1st line
					Hunk{2,0,2,1},  // add new line 2
				}},
			}},
		},
		ExpectedHalfBlameResults{
			"a1:README": {"", BlameVector{"a1","a1","a1"}},
			"b2:README": {"a1", BlameVector{"a1","b2","a1"}},
		},
	}, {
		CommitHistory{
			Commit{"a1", []FileHunks{
				FileHunks{"README", []Hunk{
					Hunk{0,0,1,3},
				}},
			}},
			Commit{"b2", []FileHunks{
				FileHunks{"README", []Hunk{
					Hunk{1,3,0,0},
				}},
			}},
		},
		ExpectedHalfBlameResults{
			"a1:README": {"", BlameVector{"a1","a1","a1"}},
			"b2:README": {"a1", BlameVector{}},
		},
	}, {
		CommitHistory{
			Commit{"a1", []FileHunks{
				FileHunks{"README", []Hunk{
					Hunk{0,0,1,3},
				}},
			}},
			Commit{"b2", []FileHunks{
				FileHunks{"hello.c", []Hunk{
					Hunk{0,0,1,2},
				}},
			}},
			Commit{"c3", []FileHunks{
				FileHunks{"README", []Hunk{
					Hunk{0,0,4,1},
				}},
			}},
		},
		ExpectedHalfBlameResults{
			"a1:README": {"", BlameVector{"a1","a1","a1"}},
			"b2:README": {"a1", BlameVector{}},
			"b2:hello.c": {"", BlameVector{"b2", "b2"}},
			"c3:README": {"a1", BlameVector{"a1","a1","a1","c3"}},
			"c3:hello.c": {"b2", BlameVector{}},
		},
	}}
	for test_number, c := range tests {
		index, _ := Build_half_index(c.input_commits, nil)
		for key, desired := range c.expected_output {
			actual := (*index)[key]
			actual_vector := flatten_segments(actual.segments)
			t1 := desired.PreviousCommit == actual.PreviousCommit
			t2 := reflect.DeepEqual(desired.Vector, actual_vector)
			if (!(t1 && t2)) {
				t.Error("Test", test_number, "key", key,
					"failed\n  Wanted", desired.Vector,
					"\n  Got   ", actual_vector,
					"\n  From  ", actual.segments)
			}
		}
	}
}

type ExpectedBlameResults map[string]BlameResult

func TestFullIndexing(t *testing.T) {
	var tests = []struct {
		input_commits CommitHistory
		expected_output ExpectedBlameResults
	}{{
		CommitHistory{},
		ExpectedBlameResults{},
	}, {
		CommitHistory{
			Commit{"a1", []FileHunks{
				FileHunks{"README", []Hunk{
					Hunk{0,0,1,2},
				}},
			}},
			Commit{"b2", []FileHunks{
				FileHunks{"README", []Hunk{
					Hunk{1,0,2,1},
				}},
			}},
			Commit{"c3", []FileHunks{
				FileHunks{"README", []Hunk{
					Hunk{3,1,3,0},
				}},
			}},
		},
		ExpectedBlameResults{
			"a1:README": {"", "b2",
				BlameVector{"a1","a1"},
				BlameVector{"","c3"}},
			"b2:README": {"a1", "c3",
				BlameVector{"a1","b2","a1"},
				BlameVector{"","","c3"}},
			"c3:README": {"b2", "",
				BlameVector{"a1","b2"},
				BlameVector{"",""}},
		},
	}, {
		CommitHistory{
			Commit{"a1", []FileHunks{
				FileHunks{"README", []Hunk{
					Hunk{0,0,1,3},
				}},
			}},
			Commit{"b2", []FileHunks{
				FileHunks{"hello.c", []Hunk{
					Hunk{0,0,1,2},
				}},
			}},
			Commit{"c3", []FileHunks{
				FileHunks{"README", []Hunk{
					Hunk{2,1,1,0},
					Hunk{3,0,3,1},
				}},
			}},
		},
		ExpectedBlameResults{
			"a1:README": {"", "c3",
				BlameVector{"a1","a1","a1"},
				BlameVector{"","c3",""}},
			// "a1:hello.c": {"", "b2"},  TODO: pre-existence pointers?
			"b2:README": {"a1", "c3",
				BlameVector{}, BlameVector{}},
			"b2:hello.c": {"", "",
				BlameVector{"b2","b2"},
				BlameVector{"",""}},
			"c3:README": {"a1", "",
				BlameVector{"a1","a1","c3"},
				BlameVector{"","",""}},
			"c3:hello.c": {"b2", "",
				BlameVector{}, BlameVector{}},
		},
	}, {
		CommitHistory{
			Commit{"a1", []FileHunks{
				FileHunks{"tools/gen_build.go", []Hunk{
					Hunk{0,0,1,3},
				}},
			}},
			Commit{"b2", []FileHunks{
				FileHunks{"tools/gen_build.go", []Hunk{
					Hunk{1,3,0,0}, // file deletion
				}},
			}},
		},
		ExpectedBlameResults{
			"a1:tools/gen_build.go": {"", "b2",
				BlameVector{"a1","a1","a1"},
				BlameVector{"b2","b2","b2"}},
			"b2:tools/gen_build.go": {"a1", "",
				BlameVector{},
				BlameVector{}},
		},
	}}
	for test_number, c := range tests {
		index := Build_index(c.input_commits)
		for key, desired_result := range c.expected_output {
			i := strings.Index(key, ":")
			commit_hash := key[:i]
			path := key[i+1:]
			actual_result, _ := index.GetNode(commit_hash, path)
			if !reflect.DeepEqual(&desired_result, actual_result) {
				t.Error("Test", test_number, "key", key,
					"failed\n  Wanted", desired_result,
					"\n  Got   ", actual_result)
			}
		}
	}
}
