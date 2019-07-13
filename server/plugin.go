package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
)

const (
	botName        = "loudspeaker"
	botDisplayName = "Loud Speaker"

	commandTrigger = "loudspeaker"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	botUserID string
}

// OnActivate activate plugin
func (p *Plugin) OnActivate() error {
	bot := &model.Bot{
		Username:    botName,
		DisplayName: botDisplayName,
	}
	botUserID, appErr := p.Helpers.EnsureBot(bot)
	if appErr != nil {
		return errors.Wrap(appErr, "failed to ensure bot user")
	}
	p.botUserID = botUserID

	// Set profile image for the bot.
	// Even if getting errors, don't stop process.
	if err := p.setProfileImage(); err != nil {
		p.API.LogWarn("failed to set profile image for the bot", "details", err.Error())
	}

	if err := p.API.RegisterCommand(&model.Command{
		Trigger:          commandTrigger,
		AutoComplete:     true,
		AutoCompleteDesc: "Announce message to all teams",
		AutoCompleteHint: "[Message]",
	}); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) setProfileImage() error {
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return err
	}
	profileImage, err := ioutil.ReadFile(filepath.Join(bundlePath, "assets", "icon.png"))
	if err != nil {
		return err
	}
	if appErr := p.API.SetProfileImage(p.botUserID, profileImage); appErr != nil {
		return errors.Wrap(appErr, "failed to set profile image for the bot")
	}
	return nil
}

// ExecuteCommand create post for the default channel of all teams
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	announcement := &announcement{
		message: strings.TrimLeft(args.Command, fmt.Sprintf("/%s ", commandTrigger)),
		userID:  args.UserId,
	}

	// TODO: Check permission
	teams, appErr := p.API.GetTeams()
	if appErr != nil {
		return nil, appErr
	}

	for _, team := range teams {
		ch, appErr := p.API.GetChannelByName(team.Id, model.DEFAULT_CHANNEL, false)
		if appErr != nil {
			announcement.results = append(announcement.results, announcementResult{
				resultType: resultErrorNotFoundDefaultChannel,
				team:       team,
			})
			continue
		}

		_, appErr = p.API.CreatePost(&model.Post{
			UserId:    p.botUserID,
			ChannelId: ch.Id,
			Message:   announcement.message,
		})
		if appErr != nil {
			announcement.results = append(announcement.results, announcementResult{
				resultType: resultErrorCreatePost,
				team:       team,
				channel:    ch,
			})
			continue
		}
		announcement.results = append(announcement.results, announcementResult{
			resultType: resultSuccess,
			team:       team,
			channel:    ch,
		})
	}

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         announcement.getResultTable(),
	}, nil
}

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, world!")
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
