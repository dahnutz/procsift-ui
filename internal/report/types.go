package report

import "time"

// MaxBytes is the upper bound for JSON report input.
const MaxBytes = 64 << 20

// Report is the operator-facing ProcSift JSON report.
type Report struct {
	Tool           string      `json:"tool"`
	Version        string      `json:"version"`
	MinSeverity    string      `json:"min_severity"`
	HostResult     string      `json:"host_result"`
	HostExitReason string      `json:"host_exit_reason"`
	HostSignals    HostSignals `json:"host_signals"`
	Time           time.Time   `json:"time"`
	DurationMS     int64       `json:"duration_ms"`
	Host           HostInfo    `json:"host"`
	ScannerPID     int         `json:"scanner_pid,omitempty"`
	ScannerTicks   uint64      `json:"scanner_start_ticks,omitempty"`
	Counts         SevCounts   `json:"counts"`
	HostCounts     SevCounts   `json:"host_counts"`
	ScannerCounts  SevCounts   `json:"scanner_counts"`
	FileChecks     []FileCheck `json:"file_checks,omitempty"`
	Attention      []Attention `json:"attention,omitempty"`
	Findings       []Finding   `json:"findings"`
	Processes      []Process   `json:"processes,omitempty"`
	Sockets        []Socket    `json:"sockets,omitempty"`
	Notes          []string    `json:"notes"`
	Limits         []string    `json:"limits"`
	Coverage       []ReadIssue `json:"coverage"`
}

type HostInfo struct {
	Hostname      string   `json:"hostname"`
	Kernel        string   `json:"kernel"`
	OSRelease     string   `json:"os_release"`
	BootID        string   `json:"boot_id,omitempty"`
	EUID          int      `json:"euid"`
	PIDMax        int      `json:"pid_max"`
	SweepUsed     bool     `json:"pid_sweep"`
	TasksSeen     int      `json:"tasks_seen"`
	SweepProbed   int      `json:"pid_sweep_probed"`
	SweepComplete bool     `json:"pid_sweep_complete"`
	Taint         uint64   `json:"taint"`
	TaintFlags    []string `json:"taint_flags,omitempty"`
}

type SevCounts struct {
	High   int `json:"high"`
	Medium int `json:"medium"`
	Low    int `json:"low"`
	Info   int `json:"info"`
}

type Process struct {
	PID           int    `json:"pid"`
	PPID          int    `json:"ppid"`
	UID           int    `json:"uid"`
	State         string `json:"state"`
	Name          string `json:"name"`
	RSSKiB        int    `json:"rss_kib"`
	Threads       int    `json:"threads"`
	Exe           string `json:"exe,omitempty"`
	Cmdline       string `json:"cmdline,omitempty"`
	Cwd           string `json:"cwd,omitempty"`
	Listed        bool   `json:"listed"`
	Kthread       bool   `json:"kthread,omitempty"`
	KthreadKnown  bool   `json:"kthread_known,omitempty"`
	StartupStatus string `json:"startup_status,omitempty"`
	StartTicks    uint64 `json:"start_ticks,omitempty"`
	ExeDev        string `json:"exe_dev,omitempty"`
	ExeInode      uint64 `json:"exe_inode,omitempty"`
	ExeHash       string `json:"exe_hash,omitempty"`
	ExeHashScope  string `json:"exe_hash_scope,omitempty"`
}

type Socket struct {
	Proto  string `json:"proto"`
	Local  string `json:"local"`
	Remote string `json:"remote"`
	State  string `json:"state"`
	Inode  uint64 `json:"inode,omitempty"`
	PID    int    `json:"pid,omitempty"`
}

type HostSignals struct {
	PreloadPaths    []string `json:"preload_paths,omitempty"`
	BPFPins         []string `json:"bpf_pins,omitempty"`
	BPFUnprivileged string   `json:"bpf_unprivileged,omitempty"`
	BPFTree         string   `json:"bpf_tree,omitempty"`
	RPM             string   `json:"rpm,omitempty"`
}

type Attention struct {
	PID          int      `json:"pid,omitempty"`
	Process      string   `json:"process,omitempty"`
	Subject      string   `json:"subject,omitempty"`
	Score        int      `json:"score"`
	Level        string   `json:"level"`
	Why          []string `json:"why"`
	Alternatives []string `json:"alternatives,omitempty"`
}

type FileCheck struct {
	Path     string `json:"path"`
	Package  string `json:"package,omitempty"`
	State    string `json:"state"`
	Source   string `json:"source,omitempty"`
	MD5      string `json:"md5,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
	Manifest string `json:"manifest,omitempty"`
	Bytes    int64  `json:"bytes,omitempty"`
}

type ReadIssue struct {
	Source string `json:"source"`
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
}

type Finding struct {
	Rule     string   `json:"rule"`
	Severity string   `json:"severity"`
	Category string   `json:"category"`
	PID      int      `json:"pid,omitempty"`
	Process  string   `json:"process,omitempty"`
	Subject  string   `json:"subject,omitempty"`
	Summary  string   `json:"summary"`
	Detail   string   `json:"detail,omitempty"`
	Evidence []string `json:"evidence,omitempty"`
	Origin   string   `json:"origin,omitempty"`
}
