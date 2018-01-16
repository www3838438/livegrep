package blameworthy

import (
//	"fmt"
//	"strings"
)

type BlameSegment struct {
	LineCount int
	CommitHash string
}

type BlameSegments []BlameSegment;

type BlameVector []string	// Blames every line on a commit hash

// func Build_index(commits CommitHistory) (*BlameIndex) {
// 	forward_index, fileLengths := Build_half_index(commits, nil)
// 	reverse_history_in_place(commits)
// 	reverse_index, _ := Build_half_index(commits, fileLengths)

// 	// TODO: avoid having to put everything back by maybe switching
// 	// to an iterator or something?
// 	reverse_history_in_place(commits)

// 	blame_index := make(BlameIndex)

// 	for key, forward_entry := range *forward_index {
// 		var reverse_segments BlameSegments

// 		i := strings.Index(key, ":")
// 		colon_path := key[i:]  // the colon followed by the path
// 		mid_entry := (*reverse_index)[key]
// 		commit_hash := mid_entry.PreviousCommit

// 		if len(forward_entry.segments) == 0 {
// 			reverse_segments = BlameSegments{}
// 		} else {
// 			key2 := commit_hash + colon_path
// 			reverse_entry, ok := (*reverse_index)[key2]
// 			if ok {
// 				reverse_segments = reverse_entry.segments
// 			} else {
// 				length := sum_lines(forward_entry.segments)
// 				reverse_segments = BlameSegments{{length, ""}}
// 			}
// 		}
// 		blame_index[key] = BlameEntry{
// 			forward_entry.PreviousCommit,
// 			commit_hash,
// 			forward_entry.segments,
// 			reverse_segments,
// 		}
// 	}
// 	return &blame_index
// }

// func (ix BlameIndex) GetNode(commit_hash string, path string,
// ) (*BlameResult, bool) {
// 	if len(commit_hash) > hashLength {
// 		commit_hash = commit_hash[:hashLength]
// 	}
// 	key := commit_hash + ":" + path
// 	entry, ok := ix[key]
// 	if !ok {
// 		return nil, false
// 	}
// 	result := BlameResult{
// 		entry.PreviousCommit,
// 		entry.NextCommit,
// 		flatten_segments(entry.Blame),
// 		flatten_segments(entry.Future),
// 	}
// 	return &result, true
// }

// func (ix BlameIndex) GetFile(commit_hash string, path string,
// ) (*BlameResult, bool) {
// 	if len(commit_hash) > hashLength {
// 		commit_hash = commit_hash[:hashLength]
// 	}
// 	key := commit_hash + ":" + path
// 	entry, ok := ix[key]
// 	fmt.Print(key, ok, "\n")
// 	if !ok {
// 		return nil, false
// 	}
// 	if len(entry.Blame) == 0 && commit_hash != entry.PreviousCommit {
// 		key := entry.PreviousCommit + ":" + path
// 		entry, ok = ix[key]
// 		if !ok {
// 			return nil, false
// 		}
// 	}
// 	result := BlameResult{
// 		entry.PreviousCommit,
// 		entry.NextCommit,
// 		flatten_segments(entry.Blame),
// 		flatten_segments(entry.Future),
// 	}
// 	return &result, true
// }

func (commit FileCommit) step(oldb BlameSegments) (BlameSegments) {
	newb := BlameSegments{}
	olineno := 1
	nlineno := 1

	oi := 0
	ocount := 0
	if len(oldb) > 0 {
		ocount = oldb[0].LineCount
	}

	// fmt.Print("===================\n")
	// fmt.Print("Initial lengths: ", initLengths, "\n")
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

	// fmt.Print("COMMIT ", commit.Hash, "\n")

	// fmt.Print("A\n")
	for _, h := range(commit.Hunks) {
		// fmt.Print("HUNK ", h, "\n")
		if h.OldLength > 0 {
			ff(h.OldStart - olineno)
			skip(h.OldLength)
		}
		if h.NewLength > 0 {
			ff(h.NewStart - nlineno)
			add(h.NewLength, commit.Hash)
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
	//fmt.Print("RETURNING: ", half_index, "\nAND: ", fileLengths, "\n")
	return newb
}

// func reverse_history_in_place(commits CommitHistory) {
// 	// Reverse the order of the commits themselves.
// 	half := len(commits) / 2
// 	last := len(commits) - 1
// 	for i := 0; i < half; i++ {
// 		commits[i], commits[last-i] =
// 			commits[last-i], commits[i]
// 	}

// 	// Reverse the effect of each hunk.
// 	for i := range commits {
// 		for j := range commits[i].Files {
// 			for k := range commits[i].Files[j].Hunks {
// 				h := &commits[i].Files[j].Hunks[k]
// 				h.OldStart, h.NewStart =
// 					h.NewStart, h.OldStart
// 				h.OldLength, h.NewLength =
// 					h.NewLength, h.OldLength
// 			}
// 		}
// 	}
// }

func sum_lines(segments BlameSegments) (int) {
	n := 0
	for _, segment := range segments {
		n += segment.LineCount
	}
	return n
}

func (segments BlameSegments) flatten() (BlameVector) {
	v := BlameVector{}
	for _, segment := range segments {
		for i := 0; i < segment.LineCount; i++ {
			v = append(v, segment.CommitHash)
		}
	}
	return v
}
