package tty

// ActionKind describes what a decoded keypress means.
type ActionKind int

const (
	ActionNone ActionKind = iota
	ActionRun
	ActionEdit
	ActionCancel
)

// Action is the result of decoding input.
type Action struct {
	Kind  ActionKind
	Index int // 0-based index into the shown list; valid for ActionRun/ActionEdit
}

// plain digit bytes are the same on every keyboard layout and terminal, unlike a shift-chord
var runBytes = [9]byte{'1', '2', '3', '4', '5', '6', '7', '8', '9'}

// Decoder turns raw input bytes into Actions. Edit is "e" then a digit (e.g. "e1"), not a
// shift-chord, so it works the same on any layout.
type Decoder struct {
	pendingEdit bool
}

func NewDecoder() *Decoder {
	return &Decoder{}
}

// Feed processes one byte and returns the resulting Action; ActionNone means keep reading.
func (d *Decoder) Feed(b byte, n int) Action {
	if d.pendingEdit {
		d.pendingEdit = false
		if b == 0x03 || b == '0' {
			return Action{Kind: ActionCancel}
		}
		if idx, ok := digitIndex(b, n); ok {
			return Action{Kind: ActionEdit, Index: idx}
		}
		return Action{Kind: ActionNone} // bad follow-up, drop the pending 'e'
	}

	if b == 0x03 || b == '0' {
		return Action{Kind: ActionCancel}
	}
	if b == 'e' || b == 'E' {
		d.pendingEdit = true
		return Action{Kind: ActionNone}
	}
	if idx, ok := digitIndex(b, n); ok {
		return Action{Kind: ActionRun, Index: idx}
	}
	return Action{Kind: ActionNone}
}

func digitIndex(b byte, n int) (int, bool) {
	if n > 9 {
		n = 9
	}
	for i := 0; i < n; i++ {
		if b == runBytes[i] {
			return i, true
		}
	}
	return 0, false
}
