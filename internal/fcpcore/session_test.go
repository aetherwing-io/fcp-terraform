package fcpcore

import (
	"fmt"
	"testing"
)

type mockModel struct {
	title string
	data  []string
}

type mockHooks struct {
	newCalled            bool
	newParams            map[string]string
	openCalled           bool
	openPath             string
	saveCalled           bool
	savePath             string
	rebuildCalled        bool
	openError            error
	saveError            error
}

func (h *mockHooks) OnNew(params map[string]string) (any, error) {
	h.newCalled = true
	h.newParams = params
	title := params["title"]
	if title == "" {
		title = "Untitled"
	}
	return &mockModel{title: title, data: []string{}}, nil
}

func (h *mockHooks) OnOpen(path string) (any, error) {
	h.openCalled = true
	h.openPath = path
	if h.openError != nil {
		return nil, h.openError
	}
	return &mockModel{title: path, data: []string{"loaded"}}, nil
}

func (h *mockHooks) OnSave(model any, path string) error {
	h.saveCalled = true
	h.savePath = path
	return h.saveError
}

func (h *mockHooks) OnRebuildIndices(model any) {
	h.rebuildCalled = true
}

func (h *mockHooks) GetDigest(model any) string {
	m := model.(*mockModel)
	return fmt.Sprintf("[%s: %d items]", m.title, len(m.data))
}

func createMockSession() (*Session, *mockHooks) {
	hooks := &mockHooks{}
	reverseCalled := false
	replayCalled := false
	session := NewSession(hooks,
		func(event any, model any) { reverseCalled = true; _ = reverseCalled },
		func(event any, model any) { replayCalled = true; _ = replayCalled },
	)
	return session, hooks
}

func TestSession_New_WithTitle(t *testing.T) {
	session, hooks := createMockSession()
	result := session.Dispatch(`new "My Song"`)
	if result != `new "My Song" created` {
		t.Errorf("result = %q, want %q", result, `new "My Song" created`)
	}
	if session.Model == nil {
		t.Fatal("model should not be nil")
	}
	m := session.Model.(*mockModel)
	if m.title != "My Song" {
		t.Errorf("title = %q, want %q", m.title, "My Song")
	}
	if !hooks.newCalled {
		t.Error("OnNew not called")
	}
	if hooks.newParams["title"] != "My Song" {
		t.Errorf("params[title] = %q", hooks.newParams["title"])
	}
}

func TestSession_New_WithParams(t *testing.T) {
	session, hooks := createMockSession()
	session.Dispatch(`new "Test" tempo:120`)
	if hooks.newParams["title"] != "Test" {
		t.Errorf("params[title] = %q", hooks.newParams["title"])
	}
	if hooks.newParams["tempo"] != "120" {
		t.Errorf("params[tempo] = %q", hooks.newParams["tempo"])
	}
}

func TestSession_New_DefaultTitle(t *testing.T) {
	session, _ := createMockSession()
	result := session.Dispatch("new")
	if result != `new "Untitled" created` {
		t.Errorf("result = %q", result)
	}
}

func TestSession_New_ClearsFilePath(t *testing.T) {
	session, _ := createMockSession()
	session.Dispatch("new Test")
	if session.FilePath != "" {
		t.Errorf("filePath = %q, want empty", session.FilePath)
	}
}

func TestSession_Open(t *testing.T) {
	session, hooks := createMockSession()
	result := session.Dispatch("open ./test.mid")
	if result != `opened "./test.mid"` {
		t.Errorf("result = %q", result)
	}
	if session.Model == nil {
		t.Fatal("model should not be nil")
	}
	if session.FilePath != "./test.mid" {
		t.Errorf("filePath = %q", session.FilePath)
	}
	if !hooks.openCalled {
		t.Error("OnOpen not called")
	}
}

func TestSession_Open_RequiresPath(t *testing.T) {
	session, _ := createMockSession()
	result := session.Dispatch("open")
	if result != "open requires a file path" {
		t.Errorf("result = %q", result)
	}
}

func TestSession_Open_Error(t *testing.T) {
	session, hooks := createMockSession()
	hooks.openError = fmt.Errorf("file not found")
	result := session.Dispatch("open ./missing.mid")
	if result != "error: file not found" {
		t.Errorf("result = %q", result)
	}
}

func TestSession_Save(t *testing.T) {
	session, hooks := createMockSession()
	session.Dispatch("open ./test.mid")
	result := session.Dispatch("save")
	if result != `saved "./test.mid"` {
		t.Errorf("result = %q", result)
	}
	if !hooks.saveCalled {
		t.Error("OnSave not called")
	}
}

func TestSession_Save_WithAs(t *testing.T) {
	session, _ := createMockSession()
	session.Dispatch("new Test")
	result := session.Dispatch("save as:./output.mid")
	if result != `saved "./output.mid"` {
		t.Errorf("result = %q", result)
	}
	if session.FilePath != "./output.mid" {
		t.Errorf("filePath = %q", session.FilePath)
	}
}

func TestSession_Save_NoModel(t *testing.T) {
	session, _ := createMockSession()
	result := session.Dispatch("save")
	if result != "error: no model to save" {
		t.Errorf("result = %q", result)
	}
}

func TestSession_Save_NoPath(t *testing.T) {
	session, _ := createMockSession()
	session.Dispatch("new Test")
	result := session.Dispatch("save")
	if result != "error: no file path. Use save as:./file" {
		t.Errorf("result = %q", result)
	}
}

func TestSession_Checkpoint(t *testing.T) {
	session, _ := createMockSession()
	result := session.Dispatch("checkpoint v1")
	if result != `checkpoint "v1" created` {
		t.Errorf("result = %q", result)
	}
	if session.Log.Cursor() == 0 {
		t.Error("cursor should be > 0 after checkpoint")
	}
}

func TestSession_Checkpoint_RequiresName(t *testing.T) {
	session, _ := createMockSession()
	result := session.Dispatch("checkpoint")
	if result != "checkpoint requires a name" {
		t.Errorf("result = %q", result)
	}
}

func TestSession_Undo(t *testing.T) {
	session, _ := createMockSession()
	session.Dispatch("new Test")
	session.Log.Append("event1")
	session.Log.Append("event2")
	result := session.Dispatch("undo")
	if result != "undone 1 event" {
		t.Errorf("result = %q", result)
	}
}

func TestSession_Undo_NoModel(t *testing.T) {
	session, _ := createMockSession()
	result := session.Dispatch("undo")
	if result != "nothing to undo" {
		t.Errorf("result = %q", result)
	}
}

func TestSession_Undo_ToCheckpoint(t *testing.T) {
	session, _ := createMockSession()
	session.Dispatch("new Test")
	session.Log.Append("event1")
	session.Dispatch("checkpoint v1")
	session.Log.Append("event2")
	session.Log.Append("event3")
	result := session.Dispatch("undo to:v1")
	if result == "" {
		t.Error("expected result")
	}
	// Should contain "undone" and "checkpoint"
	if result != `undone 2 events to checkpoint "v1"` {
		t.Errorf("result = %q", result)
	}
}

func TestSession_Undo_CallsRebuild(t *testing.T) {
	session, hooks := createMockSession()
	session.Dispatch("new Test")
	session.Log.Append("event1")
	session.Dispatch("undo")
	if !hooks.rebuildCalled {
		t.Error("OnRebuildIndices not called after undo")
	}
}

func TestSession_Redo(t *testing.T) {
	session, _ := createMockSession()
	session.Dispatch("new Test")
	session.Log.Append("event1")
	session.Log.Undo(1)
	result := session.Dispatch("redo")
	if result != "redone 1 event" {
		t.Errorf("result = %q", result)
	}
}

func TestSession_Redo_NoModel(t *testing.T) {
	session, _ := createMockSession()
	result := session.Dispatch("redo")
	if result != "nothing to redo" {
		t.Errorf("result = %q", result)
	}
}

func TestSession_Redo_CallsRebuild(t *testing.T) {
	session, hooks := createMockSession()
	session.Dispatch("new Test")
	session.Log.Append("event1")
	session.Log.Undo(1)
	session.Dispatch("redo")
	if !hooks.rebuildCalled {
		t.Error("OnRebuildIndices not called after redo")
	}
}

func TestSession_UnknownCommand(t *testing.T) {
	session, _ := createMockSession()
	result := session.Dispatch("explode")
	if result != `unknown session action "explode"` {
		t.Errorf("result = %q", result)
	}
}

func TestSession_EmptyAction(t *testing.T) {
	session, _ := createMockSession()
	result := session.Dispatch("")
	if result != "empty action" {
		t.Errorf("result = %q", result)
	}
}
