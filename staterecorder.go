package gmars

type CoreState uint8

const (
	CoreEmpty CoreState = iota
	CoreExecuted
	CoreWritten
	CoreIncremented
	CoreDecremented
	CoreRead
	CoreTerminated
)

// HistoryEntry represents a single historical event in the core
type HistoryEntry struct {
	State CoreState
	Color int
}

// StateRecorder implements a Reporter which records operations performed at each core address
// and maintains a configurable history of events
type StateRecorder struct {
	sim         ReportingSimulator
	coresize    Address
	recordReads bool

	history     [][]HistoryEntry // Slice of circular buffers, one for each address
	historySize int              // Maximum number of historical events to store per address
	historyPos  []int            // Current position in each circular buffer
}

// NewStateRecorder creates a new StateRecorder with configurable history size
func NewStateRecorder(sim ReportingSimulator, historySize int) *StateRecorder {
	coresize := sim.CoreSize()

	// Initialize history-related structures
	history := make([][]HistoryEntry, coresize)
	historyPos := make([]int, coresize)

	for i := Address(0); i < coresize; i++ {
		history[i] = make([]HistoryEntry, historySize)
		// Initialize with empty state
		for j := 0; j < historySize; j++ {
			history[i][j] = HistoryEntry{
				State: CoreEmpty,
				Color: -1,
			}
		}
	}

	return &StateRecorder{
		sim:         sim,
		coresize:    coresize,
		history:     history,
		historySize: historySize,
		historyPos:  historyPos,
	}
}

func (r *StateRecorder) GetMemStateN(a Address, n int) (CoreState, int) {
	pos := (r.historyPos[a] - (n + 1) + r.historySize) % r.historySize
	entry := r.history[a][pos]
	return entry.State, entry.Color
}

// GetMemState returns the current state and color of a memory address
// by looking at the most recent history entry
func (r *StateRecorder) GetMemState(a Address) (CoreState, int) {
	return r.GetMemStateN(a, 0)
}

// GetHistory returns the history of events for a given address
// Returns slice of HistoryEntry ordered from most recent to oldest
func (r *StateRecorder) GetHistory(a Address) []HistoryEntry {
	result := make([]HistoryEntry, 0, r.historySize)

	// Start from most recent entry and work backwards
	pos := r.historyPos[a]
	addressHistory := r.history[a]

	// First add entries from current position back to start
	for i := pos - 1; i >= 0; i-- {
		if addressHistory[i].Color != -1 || addressHistory[i].State != CoreEmpty {
			result = append(result, addressHistory[i])
		}
	}

	// Then add entries from end of buffer to current position
	for i := r.historySize - 1; i >= pos; i-- {
		if addressHistory[i].Color != -1 || addressHistory[i].State != CoreEmpty {
			result = append(result, addressHistory[i])
		}
	}

	return result
}

func (r *StateRecorder) SetRecordRead(val bool) {
	r.recordReads = val
}

// recordHistory adds a new entry to the history for the given address
func (r *StateRecorder) recordHistory(addr Address, state CoreState, color int, report Report) {
	entry := HistoryEntry{
		State: state,
		Color: color,
	}

	// Add to circular buffer
	r.history[addr][r.historyPos[addr]] = entry
	r.historyPos[addr] = (r.historyPos[addr] + 1) % r.historySize
}

func (r *StateRecorder) reset() {
	for i := Address(0); i < r.coresize; i++ {
		// Reset history position
		r.historyPos[i] = 0
		// Clear history with empty state
		for j := 0; j < r.historySize; j++ {
			r.history[i][j] = HistoryEntry{
				State: CoreEmpty,
				Color: -1,
			}
		}
	}
}

func (r *StateRecorder) Report(report Report) {
	var newState CoreState
	var affected Address

	switch report.Type {
	case SimReset:
		r.reset()
		return

	case WarriorSpawn:
		w := r.sim.GetWarrior(report.WarriorIndex)
		for i := report.Address; i < report.Address+Address(w.Length()); i++ {
			affected = i % r.coresize
			r.recordHistory(affected, CoreWritten, report.WarriorIndex, report)
		}
		return

	case WarriorTaskTerminate:
		newState = CoreTerminated
	case WarriorTaskPop:
		newState = CoreExecuted
	case WarriorWrite:
		newState = CoreWritten
	case WarriorRead:
		if !r.recordReads {
			return
		}
		newState = CoreRead
	case WarriorIncrement:
		newState = CoreIncremented
	case WarriorDecrement:
		newState = CoreDecremented
	default:
		return
	}

	affected = report.Address % r.coresize
	r.recordHistory(affected, newState, report.WarriorIndex, report)
}
