package main

import (
	"io/ioutil"
	"path/filepath"
	"sync"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"

	"github.com/pkg/errors"
)

const (
	botUserName    = "mbotc"
	botDisplayName = "MBotC"
	botDescription = "Created by the MBotC plugin."
	clientUrl      = "https://www.mbotc.com"
	serviceAPIUrl  = "http://www.mbotc.com:8080"
	pluginId       = "com.mattermost.plugin-mbotc"
	timezone       = "Asia/Seoul"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// botUserID of the created bot account.
	botUserID string

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}


// OnActivate checks if the configurations is valid and ensures the bot account exists
func (p *Plugin) OnActivate() error {
	if p.API.GetConfig().ServiceSettings.SiteURL == nil {
		return errors.New("We couldn't find a siteURL. Please set a siteURL and restart the plugin")
	}

	botUserID, err := p.Helpers.EnsureBot(&model.Bot{
		Username:    botUserName,
		DisplayName: botDisplayName,
		Description: botDescription,
	})

	if err != nil {
		return errors.Wrap(err, "failed to create bot account")
	}
	p.botUserID = botUserID
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return errors.Wrap(err, "couldn't get bundle path")
	}

	profileImage, err := ioutil.ReadFile(filepath.Join(bundlePath, "assets", "profile.png"))
	if err != nil {
		return errors.Wrap(err, "couldn't read profile image")
	}

	if appErr := p.API.SetProfileImage(botUserID, profileImage); appErr != nil {
		return errors.Wrap(appErr, "couldn't set profile image")
	}

	command, err := p.getCommand()
	if err != nil {
		return errors.Wrap(err, "failed to get command")
	}

	err = p.API.RegisterCommand(command)
	if err != nil {
		return errors.WithMessage(err, "OnActivate: failed to register command")
	}

	return nil
}
