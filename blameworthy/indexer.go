package blameworthy

import (
	"fmt"
	"strings"
)

type BlameSegment struct {
	LineCount int
	CommitHash string
}

type BlameSegments []BlameSegment;

type HalfEntry struct {		// For a specific path and commit
	PreviousCommit string
	segments BlameSegments
}

type HalfIndex map[string]HalfEntry;

type BlameEntry struct {	// For a specific path and commit
	PreviousCommit string
	NextCommit string
	Blame BlameSegments
	Future BlameSegments
}

type BlameIndex map[string]BlameEntry;

type BlameVector []string	// Blames every line on a commit hash

type BlameResult struct {
	PreviousCommit string
	NextCommit string
	Blame BlameVector
	Future BlameVector
}

func Build_index(commits CommitHistory) (*BlameIndex) {
	forward_index, file_blames := build_half_index(commits, nil)
	reverse_history_in_place(commits)
	reverse_index, _ := build_half_index(commits, file_blames)
	// TODO: avoid having to put everything back by maybe switching
	// to an iterator or something?
	reverse_history_in_place(commits)

	blame_index := make(BlameIndex)

	for key, forward_entry := range *forward_index {
		var reverse_segments BlameSegments

		i := strings.Index(key, ":")
		colon_path := key[i:]  // the colon followed by the path
		mid_entry := (*reverse_index)[key]
		commit_hash := mid_entry.PreviousCommit

		if len(forward_entry.segments) == 0 {
			reverse_segments = BlameSegments{}
		} else {
			key2 := commit_hash + colon_path
			reverse_entry, ok := (*reverse_index)[key2]
			if ok {
				reverse_segments = reverse_entry.segments
			} else {
				length := sum_lines(forward_entry.segments)
				reverse_segments = BlameSegments{{length, ""}}
			}
		}
		blame_index[key] = BlameEntry{
			forward_entry.PreviousCommit,
			commit_hash,
			forward_entry.segments,
			reverse_segments,
		}
	}
	return &blame_index
}

func (ix BlameIndex) GetNode(commit_hash string, path string,
) (*BlameResult, bool) {
	if len(commit_hash) > hashLength {
		commit_hash = commit_hash[:hashLength]
	}
	key := commit_hash + ":" + path
	entry, ok := ix[key]
	if !ok {
		return nil, false
	}
	result := BlameResult{
		entry.PreviousCommit,
		entry.NextCommit,
		flatten_segments(entry.Blame),
		flatten_segments(entry.Future),
	}
	return &result, true
}

func (ix BlameIndex) GetFile(commit_hash string, path string,
) (*BlameResult, bool) {
	if len(commit_hash) > hashLength {
		commit_hash = commit_hash[:hashLength]
	}
	key := commit_hash + ":" + path
	entry, ok := ix[key]
	fmt.Print(key, ok, "\n")
	if !ok {
		return nil, false
	}
	if len(entry.Blame) == 0 && commit_hash != entry.PreviousCommit {
		key := entry.PreviousCommit + ":" + path
		entry, ok = ix[key]
		if !ok {
			return nil, false
		}
	}
	result := BlameResult{
		entry.PreviousCommit,
		entry.NextCommit,
		flatten_segments(entry.Blame),
		flatten_segments(entry.Future),
	}
	return &result, true
}

// Private helpers

func build_half_index(
	commits CommitHistory,
	init_lengths *map[string]int,
) (*HalfIndex, *map[string]int) {
	half_index := make(HalfIndex)
	file_blames := make(map[string]BlameSegments)
	file_commit_hashes := make(map[string]string)

	if init_lengths != nil {
		for path, length := range *init_lengths {
			file_blames[path] = BlameSegments{{length, ""}}
			file_commit_hashes[path] = ""
		}
	}

	var ok bool
	var olineno int		// line numbers, so 1-based, not 0-based
	var nlineno int
	var newb BlameSegments
	var oldb BlameSegments

	oi := 0
	ocount := 0

	// fmt.Print("===================\n")
	// fmt.Print("Initial lengths: ", init_lengths, "\n")
	// fmt.Print("Initial blames: ", file_blames, "\n")

	ff := func(linecount int) {
		// fmt.Print("ff ", linecount, "\n")
		for linecount > 0 && linecount >= ocount {
			// fmt.Print(linecount, oldb, oi, "\n")
			commit_hash := oldb[oi].CommitHash
			newb = append(newb, BlameSegment{ocount, commit_hash})
			nlineno += ocount
			linecount -= ocount
			olineno += ocount
			oi += 1
			ocount = 0
			if oi < len(oldb) {
				ocount = oldb[oi].LineCount
			}
		}
		if linecount > 0 {
			commit_hash := oldb[oi].CommitHash
			newb = append(newb,
				BlameSegment{linecount, commit_hash})
			nlineno += linecount
			ocount -= linecount
			olineno += linecount
		}
		// for i := 0; i < linecount; i++ {
		// 	newb = append(newb, oldb[olineno - 1])
		// 	olineno += 1
		// 	nlineno += 1
		// }
	}
	skip := func(linecount int) {
		// fmt.Print("skip ", linecount, ocount, oi, oldb, "\n")
		for linecount > 0 && linecount >= ocount {
			linecount -= ocount
			olineno += ocount
			oi += 1
			ocount = 0
			if oi < len(oldb) {
				ocount = oldb[oi].LineCount
			}
		}
		ocount -= linecount
		olineno += linecount
		// olineno += linecount
		// fmt.Print("skip done")
	}
	add := func(linecount int, commit_hash string) {
		// fmt.Print("add ", linecount, commit_hash, "\n")
		newb = append(newb, BlameSegment{linecount, commit_hash})
		nlineno += linecount
	}

	for _, commit := range commits {
		// fmt.Print("COMMIT ", commit.Hash, "\n")
		// Each unchanged file will keep existing in the next
		// revision, so this first loop creates a pointer for
		// every file back to its most recent modification.  The
		// next loop will overwrite the pointers for files
		// changed in this revision.
		for path, commit_hash := range file_commit_hashes {
			key := fmt.Sprintf("%s:%s", commit.Hash, path)
			half_index[key] = HalfEntry{commit_hash,
				BlameSegments{}}
		}

		for _, file := range commit.Files {
			// fmt.Print("PATH ", file.Path, "\n")
			oldb, ok = file_blames[file.Path]
			if !ok {
				oldb = BlameSegments{}
			}

			previous_commit := file_commit_hashes[file.Path]
			newb = BlameSegments{}

			olineno = 1
			nlineno = 1

			oi = 0
			ocount = 0
			if len(oldb) > 0 {
				ocount = oldb[0].LineCount
			}

			// fmt.Print("A\n")
			for _, h := range(file.Hunks) {
				// fmt.Print("HUNK ", h, "\n")
				if h.Old_length > 0 {
					ff(h.Old_start - olineno)
					skip(h.Old_length)
				}
				if h.New_length > 0 {
					ff(h.New_start - nlineno)
					add(h.New_length, commit.Hash)
				}
			}
			// fmt.Print("B\n")
			for oi < len(oldb) {
				// fmt.Print("Trying to ff", ocount, "\n")
				if ocount > 0 {
					ff(ocount)
				} else {
					oi += 1
					ocount = 0
					if oi < len(oldb) {
						ocount = oldb[oi].LineCount
					}
				}
			}
			// fmt.Print("C\n")
                        key := fmt.Sprintf("%s:%s", commit.Hash, file.Path)
			half_index[key] = HalfEntry{previous_commit, newb}

			file_blames[file.Path] = newb
			file_commit_hashes[file.Path] = commit.Hash
		}
	}

	file_lengths := make(map[string]int)
	for path, segments := range file_blames {
		file_lengths[path] = sum_lines(segments)
	}

	//fmt.Print("RETURNING: ", half_index, "\nAND: ", file_lengths, "\n")
	return &half_index, &file_lengths
}

func reverse_history_in_place(commits CommitHistory) {
	// Reverse the order of the commits themselves.
	half := len(commits) / 2
	last := len(commits) - 1
	for i := 0; i < half; i++ {
		commits[i], commits[last-i] =
			commits[last-i], commits[i]
	}

	// Reverse the effect of each hunk.
	for i := range commits {
		for j := range commits[i].Files {
			for k := range commits[i].Files[j].Hunks {
				h := &commits[i].Files[j].Hunks[k]
				h.Old_start, h.New_start =
					h.New_start, h.Old_start
				h.Old_length, h.New_length =
					h.New_length, h.Old_length
			}
		}
	}
}

func sum_lines(segments BlameSegments) (int) {
	n := 0
	for _, segment := range segments {
		n += segment.LineCount
	}
	return n
}

func flatten_segments(segments BlameSegments) (BlameVector) {
	v := BlameVector{}
	for _, segment := range segments {
		for i := 0; i < segment.LineCount; i++ {
			v = append(v, segment.CommitHash)
		}
	}
	return v
}
