package tui

import "github.com/Dicklesworthstone/coding_agent_account_manager/internal/watcher"

type profilesChangedMsg struct {
	event watcher.Event
}

type watcherReadyMsg struct {
	watcher *watcher.Watcher
	err     error
}

type badgeExpiredMsg struct {
	key string
}

func eventTypeVerb(t watcher.EventType) string {
	switch t {
	case watcher.EventProfileAdded:
		return "added"
	case watcher.EventProfileDeleted:
		return "deleted"
	default:
		return "updated"
	}
}
