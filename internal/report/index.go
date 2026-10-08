package report

import (
	"sort"
	"strconv"
)

const (
	SevInfo   = "info"
	SevLow    = "low"
	SevMedium = "medium"
	SevHigh   = "high"
)

// SeverityRank orders info < low < medium < high.
func SeverityRank(s string) int {
	switch s {
	case SevInfo:
		return 0
	case SevLow:
		return 1
	case SevMedium:
		return 2
	case SevHigh:
		return 3
	default:
		return -1
	}
}

// IndexedReport holds sorted and indexed views of one report.
type IndexedReport struct {
	Report Report

	AllFindings      []Finding
	HostFindings     []Finding
	ScannerFindings  []Finding
	AttentionByPID   map[int]int
	AttentionBySubj  map[string]int
	ProcessByPID     map[int]Process
	SortedProcesses  []Process
	SocketsByPID     map[int][]Socket
	FindingsByPID    map[int][]Finding
}

// Index builds lookup tables and sorted lists from a report.
func Index(rep Report) *IndexedReport {
	idx := &IndexedReport{
		Report:          rep,
		AttentionByPID:  make(map[int]int),
		AttentionBySubj: make(map[string]int),
		ProcessByPID:    make(map[int]Process),
		SocketsByPID:    make(map[int][]Socket),
		FindingsByPID:   make(map[int][]Finding),
	}
	for _, a := range rep.Attention {
		if a.PID > 0 {
			if prev, ok := idx.AttentionByPID[a.PID]; !ok || a.Score > prev {
				idx.AttentionByPID[a.PID] = a.Score
			}
		}
		if a.Subject != "" {
			if prev, ok := idx.AttentionBySubj[a.Subject]; !ok || a.Score > prev {
				idx.AttentionBySubj[a.Subject] = a.Score
			}
		}
	}
	idx.AllFindings = append([]Finding(nil), rep.Findings...)
	sortFindings(idx.AllFindings, idx)
	for _, f := range idx.AllFindings {
		if f.Origin == "scanner_self" {
			idx.ScannerFindings = append(idx.ScannerFindings, f)
		} else {
			idx.HostFindings = append(idx.HostFindings, f)
		}
		if f.PID > 0 {
			idx.FindingsByPID[f.PID] = append(idx.FindingsByPID[f.PID], f)
		}
	}
	idx.SortedProcesses = append([]Process(nil), rep.Processes...)
	sort.Slice(idx.SortedProcesses, func(i, j int) bool {
		return idx.SortedProcesses[i].PID < idx.SortedProcesses[j].PID
	})
	for _, p := range rep.Processes {
		idx.ProcessByPID[p.PID] = p
	}
	for _, s := range rep.Sockets {
		if s.PID > 0 {
			idx.SocketsByPID[s.PID] = append(idx.SocketsByPID[s.PID], s)
		}
	}
	return idx
}

func sortFindings(findings []Finding, idx *IndexedReport) {
	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		ar, br := SeverityRank(a.Severity), SeverityRank(b.Severity)
		if ar != br {
			return ar > br
		}
		as, bs := attentionScore(a, idx), attentionScore(b, idx)
		if as != bs {
			return as > bs
		}
		if a.PID != b.PID {
			return a.PID < b.PID
		}
		return a.Rule < b.Rule
	})
}

func attentionScore(f Finding, idx *IndexedReport) int {
	if f.PID > 0 {
		if s, ok := idx.AttentionByPID[f.PID]; ok {
			return s
		}
	}
	if f.Subject != "" {
		if s, ok := idx.AttentionBySubj[f.Subject]; ok {
			return s
		}
	}
	return -1
}

// FindingSubject returns the PID or file subject string for list display.
func FindingSubject(f Finding) string {
	if f.PID > 0 {
		return strconv.Itoa(f.PID)
	}
	if f.Subject != "" {
		return f.Subject
	}
	return "-"
}
