package fcpcore

import (
	"testing"
)

func TestEventLog_AppendAndCursor(t *testing.T) {
	log := NewEventLog()
	log.Append("a")
	log.Append("b")
	if log.Cursor() != 2 {
		t.Errorf("cursor = %d, want 2", log.Cursor())
	}
	if log.Length() != 2 {
		t.Errorf("length = %d, want 2", log.Length())
	}
}

func TestEventLog_Recent(t *testing.T) {
	log := NewEventLog()
	log.Append("a")
	log.Append("b")
	log.Append("c")

	got := log.Recent(2)
	if len(got) != 2 || got[0] != "b" || got[1] != "c" {
		t.Errorf("Recent(2) = %v, want [b c]", got)
	}

	all := log.Recent(0)
	if len(all) != 3 || all[0] != "a" || all[1] != "b" || all[2] != "c" {
		t.Errorf("Recent(0) = %v, want [a b c]", all)
	}
}

func TestEventLog_Undo_MostRecent(t *testing.T) {
	log := NewEventLog()
	log.Append("a")
	log.Append("b")
	undone := log.Undo(1)
	if len(undone) != 1 || undone[0] != "b" {
		t.Errorf("Undo(1) = %v, want [b]", undone)
	}
	if log.Cursor() != 1 {
		t.Errorf("cursor = %d, want 1", log.Cursor())
	}
}

func TestEventLog_Undo_Multiple(t *testing.T) {
	log := NewEventLog()
	log.Append("a")
	log.Append("b")
	log.Append("c")
	undone := log.Undo(2)
	if len(undone) != 2 || undone[0] != "c" || undone[1] != "b" {
		t.Errorf("Undo(2) = %v, want [c b]", undone)
	}
	if log.Cursor() != 1 {
		t.Errorf("cursor = %d, want 1", log.Cursor())
	}
}

func TestEventLog_Undo_Empty(t *testing.T) {
	log := NewEventLog()
	undone := log.Undo(1)
	if len(undone) != 0 {
		t.Errorf("Undo(empty) = %v, want []", undone)
	}
}

func TestEventLog_Undo_SkipsCheckpoints(t *testing.T) {
	log := NewEventLog()
	log.Append("a")
	log.Checkpoint("cp1")
	log.Append("b")
	undone := log.Undo(2)
	if len(undone) != 2 || undone[0] != "b" || undone[1] != "a" {
		t.Errorf("Undo(2 with checkpoint) = %v, want [b a]", undone)
	}
}

func TestEventLog_Redo(t *testing.T) {
	log := NewEventLog()
	log.Append("a")
	log.Append("b")
	log.Undo(2)
	redone := log.Redo(2)
	if len(redone) != 2 || redone[0] != "a" || redone[1] != "b" {
		t.Errorf("Redo(2) = %v, want [a b]", redone)
	}
	if log.Cursor() != 2 {
		t.Errorf("cursor = %d, want 2", log.Cursor())
	}
}

func TestEventLog_Redo_Empty(t *testing.T) {
	log := NewEventLog()
	log.Append("a")
	redone := log.Redo(1)
	if len(redone) != 0 {
		t.Errorf("Redo(at end) = %v, want []", redone)
	}
}

func TestEventLog_Redo_SkipsCheckpoints(t *testing.T) {
	log := NewEventLog()
	log.Append("a")
	log.Checkpoint("cp1")
	log.Append("b")
	log.Undo(2)
	redone := log.Redo(2)
	if len(redone) != 2 || redone[0] != "a" || redone[1] != "b" {
		t.Errorf("Redo(2 with checkpoint) = %v, want [a b]", redone)
	}
}

func TestEventLog_TruncateOnAppend(t *testing.T) {
	log := NewEventLog()
	log.Append("a")
	log.Append("b")
	log.Undo(1)   // cursor at 1, "b" is in redo tail
	log.Append("c") // should truncate "b"
	if log.Length() != 2 {
		t.Errorf("length = %d, want 2", log.Length())
	}
	redone := log.Redo(1)
	if len(redone) != 0 {
		t.Errorf("Redo after truncation = %v, want []", redone)
	}
	all := log.Recent(0)
	if len(all) != 2 || all[0] != "a" || all[1] != "c" {
		t.Errorf("Recent = %v, want [a c]", all)
	}
}

func TestEventLog_Checkpoint_UndoTo(t *testing.T) {
	log := NewEventLog()
	log.Append("a")
	log.Checkpoint("v1")
	log.Append("b")
	log.Append("c")
	undone, err := log.UndoTo("v1")
	if err != nil {
		t.Fatalf("UndoTo(v1) error: %v", err)
	}
	if len(undone) != 2 || undone[0] != "c" || undone[1] != "b" {
		t.Errorf("UndoTo(v1) = %v, want [c b]", undone)
	}
	recent := log.Recent(0)
	if len(recent) != 1 || recent[0] != "a" {
		t.Errorf("Recent after UndoTo = %v, want [a]", recent)
	}
}

func TestEventLog_Checkpoint_UnknownName(t *testing.T) {
	log := NewEventLog()
	log.Append("a")
	_, err := log.UndoTo("nonexistent")
	if err == nil {
		t.Error("expected error for unknown checkpoint")
	}
}

func TestEventLog_Checkpoint_RemovedOnTruncation(t *testing.T) {
	log := NewEventLog()
	log.Append("a")
	log.Checkpoint("v1")
	log.Append("b")
	log.Undo(2) // undo b and a, cursor before checkpoint
	log.Append("x") // truncates everything including checkpoint
	_, err := log.UndoTo("v1")
	if err == nil {
		t.Error("expected error: checkpoint should be removed after truncation")
	}
}

func TestEventLog_CursorStartsAtZero(t *testing.T) {
	log := NewEventLog()
	if log.Cursor() != 0 {
		t.Errorf("cursor = %d, want 0", log.Cursor())
	}
}

func TestEventLog_CursorAdvancesOnAppend(t *testing.T) {
	log := NewEventLog()
	log.Append("a")
	if log.Cursor() != 1 {
		t.Errorf("cursor = %d, want 1", log.Cursor())
	}
	log.Append("b")
	if log.Cursor() != 2 {
		t.Errorf("cursor = %d, want 2", log.Cursor())
	}
}

func TestEventLog_CursorMovesBackOnUndo(t *testing.T) {
	log := NewEventLog()
	log.Append("a")
	log.Append("b")
	log.Undo(1)
	if log.Cursor() != 1 {
		t.Errorf("cursor = %d, want 1", log.Cursor())
	}
}

func TestEventLog_CursorMovesForwardOnRedo(t *testing.T) {
	log := NewEventLog()
	log.Append("a")
	log.Append("b")
	log.Undo(1)
	log.Redo(1)
	if log.Cursor() != 2 {
		t.Errorf("cursor = %d, want 2", log.Cursor())
	}
}

func TestEventLog_CanUndo(t *testing.T) {
	log := NewEventLog()
	if log.CanUndo() {
		t.Error("CanUndo() should be false for empty log")
	}
	log.Append("a")
	if !log.CanUndo() {
		t.Error("CanUndo() should be true after append")
	}
}

func TestEventLog_CanRedo(t *testing.T) {
	log := NewEventLog()
	if log.CanRedo() {
		t.Error("CanRedo() should be false for empty log")
	}
	log.Append("a")
	if log.CanRedo() {
		t.Error("CanRedo() should be false at end")
	}
	log.Undo(1)
	if !log.CanRedo() {
		t.Error("CanRedo() should be true after undo")
	}
}
