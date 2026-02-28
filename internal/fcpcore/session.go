package fcpcore

import "fmt"

// SessionHooks defines the lifecycle hooks that a domain must implement.
type SessionHooks interface {
	// OnNew creates a new empty model with the given params.
	OnNew(params map[string]string) (any, error)
	// OnOpen opens a model from a file path.
	OnOpen(path string) (any, error)
	// OnSave saves a model to a file path.
	OnSave(model any, path string) error
	// OnRebuildIndices rebuilds derived indices after undo/redo.
	OnRebuildIndices(model any)
	// GetDigest returns a compact digest string for drift detection.
	GetDigest(model any) string
}

// ReverseFunc reverses a single event on the model (for undo).
type ReverseFunc func(event any, model any)

// ReplayFunc replays a single event on the model (for redo).
type ReplayFunc func(event any, model any)

// Session routes session-level actions (new, open, save, checkpoint, undo, redo)
// to the appropriate handler.
type Session struct {
	Model    any
	FilePath string
	Log      *EventLog
	hooks    SessionHooks
	reverse  ReverseFunc
	replay   ReplayFunc
}

// NewSession creates a new Session with the given hooks and event functions.
func NewSession(hooks SessionHooks, reverse ReverseFunc, replay ReplayFunc) *Session {
	return &Session{
		Log:     NewEventLog(),
		hooks:   hooks,
		reverse: reverse,
		replay:  replay,
	}
}

// Dispatch parses and executes a session command string.
// Commands: new "Title" [params...], open PATH, save, save as:PATH,
// checkpoint NAME, undo, undo to:NAME, redo
func (s *Session) Dispatch(action string) string {
	tokens := Tokenize(action)
	if len(tokens) == 0 {
		return "empty action"
	}

	cmd := tokens[0]
	// Lowercase the command
	lower := ""
	for _, c := range cmd {
		if c >= 'A' && c <= 'Z' {
			lower += string(c + 32)
		} else {
			lower += string(c)
		}
	}

	switch lower {
	case "new":
		return s.dispatchNew(tokens)
	case "open":
		return s.dispatchOpen(tokens)
	case "save":
		return s.dispatchSave(tokens)
	case "checkpoint":
		return s.dispatchCheckpoint(tokens)
	case "undo":
		return s.dispatchUndo(tokens)
	case "redo":
		return s.dispatchRedo()
	default:
		return fmt.Sprintf("unknown session action %q", lower)
	}
}

func (s *Session) dispatchNew(tokens []string) string {
	params := map[string]string{}
	var positionals []string
	for i := 1; i < len(tokens); i++ {
		if IsKeyValue(tokens[i]) {
			key, value := ParseKeyValue(tokens[i])
			params[key] = value
		} else {
			positionals = append(positionals, tokens[i])
		}
	}
	if len(positionals) > 0 {
		params["title"] = positionals[0]
	}

	model, err := s.hooks.OnNew(params)
	if err != nil {
		return fmt.Sprintf("error: %s", err.Error())
	}
	s.Model = model
	s.FilePath = ""

	title := params["title"]
	if title == "" {
		title = "Untitled"
	}
	return fmt.Sprintf("new %q created", title)
}

func (s *Session) dispatchOpen(tokens []string) string {
	if len(tokens) < 2 {
		return "open requires a file path"
	}
	path := tokens[1]
	model, err := s.hooks.OnOpen(path)
	if err != nil {
		return fmt.Sprintf("error: %s", err.Error())
	}
	s.Model = model
	s.FilePath = path
	return fmt.Sprintf("opened %q", path)
}

func (s *Session) dispatchSave(tokens []string) string {
	if s.Model == nil {
		return "error: no model to save"
	}
	savePath := s.FilePath
	for i := 1; i < len(tokens); i++ {
		if IsKeyValue(tokens[i]) {
			key, value := ParseKeyValue(tokens[i])
			if key == "as" {
				savePath = value
			}
		}
	}
	if savePath == "" {
		return "error: no file path. Use save as:./file"
	}
	err := s.hooks.OnSave(s.Model, savePath)
	if err != nil {
		return fmt.Sprintf("error: %s", err.Error())
	}
	s.FilePath = savePath
	return fmt.Sprintf("saved %q", savePath)
}

func (s *Session) dispatchCheckpoint(tokens []string) string {
	if len(tokens) < 2 {
		return "checkpoint requires a name"
	}
	name := tokens[1]
	s.Log.Checkpoint(name)
	return fmt.Sprintf("checkpoint %q created", name)
}

func (s *Session) dispatchUndo(tokens []string) string {
	if s.Model == nil {
		return "nothing to undo"
	}

	// undo to:NAME
	if len(tokens) >= 2 {
		t := tokens[1]
		if len(t) > 3 && t[:3] == "to:" {
			name := t[3:]
			if name == "" {
				return "undo to: requires a checkpoint name"
			}
			events, err := s.Log.UndoTo(name)
			if err != nil {
				return fmt.Sprintf("cannot undo to %q", name)
			}
			for _, ev := range events {
				s.reverse(ev, s.Model)
			}
			s.hooks.OnRebuildIndices(s.Model)
			return fmt.Sprintf("undone %d event%s to checkpoint %q", len(events), plural(len(events)), name)
		}
	}

	events := s.Log.Undo(1)
	if len(events) == 0 {
		return "nothing to undo"
	}
	for _, ev := range events {
		s.reverse(ev, s.Model)
	}
	s.hooks.OnRebuildIndices(s.Model)
	return fmt.Sprintf("undone %d event%s", len(events), plural(len(events)))
}

func (s *Session) dispatchRedo() string {
	if s.Model == nil {
		return "nothing to redo"
	}
	events := s.Log.Redo(1)
	if len(events) == 0 {
		return "nothing to redo"
	}
	for _, ev := range events {
		s.replay(ev, s.Model)
	}
	s.hooks.OnRebuildIndices(s.Model)
	return fmt.Sprintf("redone %d event%s", len(events), plural(len(events)))
}

func plural(n int) string {
	if n != 1 {
		return "s"
	}
	return ""
}
