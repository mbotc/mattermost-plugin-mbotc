package main

import (
	"net/http"

	"github.com/mattermost/mattermost-server/v5/plugin"
)

const (
	routeAPICreateNotificationWithCommand = "/api/v1/create-notification-with-command"
	routeAPICreateNotificationWithEditor  = "/api/v1/create-notification-with-editor"
	routeAPICreateNotificationWithButton  = "/api/v1/create-notification-with-button"
)

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch path {
	// Notification APIs
	case routeAPICreateNotificationWithCommand:
		p.httpCreateNotificationWithCommand(w, r)
	case routeAPICreateNotificationWithEditor:
		p.httpCreateNotificationWithEditor(w, r)
	case routeAPICreateNotificationWithButton:
		p.httpCreateNotificationWithButton(r)

	default:
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
}
