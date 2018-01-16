package blameworthy

import (
//	"fmt"
//	"strings"
)

type BlameSegment struct {
	LineCount int
	LineStart int
	CommitHash string
}

type BlameSegments []BlameSegment;

type BlameVector []string	// Blames every line on a commit hash

func (history FileHistory) At(index int) (BlameVector, BlameVector) {
	segments := BlameSegments{}
	var i int
	for i = 0; i <= index; i++ {
		commit := history[i]
		segments = commit.step(segments)
	}
	blameVector := segments.flatten()
	for ; i < len(history); i++ {
		commit := history[i]
		segments = commit.step(segments)
	}
	segments = segments.wipe()
	history.reverse_in_place()
	for i--; i > index; i-- {
		commit := history[i]
		segments = commit.step(segments)
	}
	futureVector := segments.flatten()
	history.reverse_in_place()
	return blameVector, futureVector
}

func (commit FileCommit) step(oldb BlameSegments) (BlameSegments) {
	newb := BlameSegments{}
	olineno := 1
	nlineno := 1

	oi := 0
	ocount := 0
	if len(oldb) > 0 {
		ocount = oldb[0].LineCount
	}

	ff := func(linecount int) {
		// fmt.Print("ff ", linecount, "\n")
		for linecount > 0 && linecount >= ocount {
			// fmt.Print(linecount, oldb, oi, "\n")
			progress := oldb[oi].LineCount - ocount
			start := oldb[oi].LineStart + progress
			hash := oldb[oi].CommitHash
			newb = append(newb, BlameSegment{ocount, start, hash})
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
			progress := oldb[oi].LineCount - ocount
			start := oldb[oi].LineStart + progress
			commit_hash := oldb[oi].CommitHash
			newb = append(newb,
				BlameSegment{linecount, start, commit_hash})
			nlineno += linecount
			ocount -= linecount
			olineno += linecount
		}
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
		start := nlineno
		newb = append(newb, BlameSegment{linecount, start, commit_hash})
		nlineno += linecount
	}

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

	return newb
}

func (commits FileHistory) reverse_in_place() {
	// Reverse the effect of each hunk.
	for i := range commits {
		for j := range commits[i].Hunks {
			h := &commits[i].Hunks[j]
			h.OldStart, h.NewStart = h.NewStart, h.OldStart
			h.OldLength, h.NewLength = h.NewLength, h.OldLength
		}
	}
}

func (segments BlameSegments) wipe() (BlameSegments) {
	n := 0
	for _, segment := range segments {
		n += segment.LineCount
	}
	return BlameSegments{{n, 1, ""}}
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
