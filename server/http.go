package main

import (
	"net/http"

	"github.com/mattermost/mattermost-server/v5/plugin"
)

const (
	routeAPICreateNoticeWithCommand = "/api/v1/create-notice-with-command"
	routeAPICreateNoticeWithEditor  = "/api/v1/create-notice-with-editor"
	routeAPICreateNoticeWithButton  = "/api/v1/create-notice-with-button"
)

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch path {
	// Notice APIs
	case routeAPICreateNoticeWithCommand:
		p.httpCreateNoticeWithCommand(w, r)
	case routeAPICreateNoticeWithEditor:
		p.httpCreateNoticeWithEditor(w, r)
	case routeAPICreateNoticeWithButton:
		p.httpCreateNoticeWithButton(w, r)

	default:
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
}
