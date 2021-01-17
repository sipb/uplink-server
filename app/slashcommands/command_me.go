// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package slashcommands

import (
	goi18n "github.com/mattermost/go-i18n/i18n"
	"github.com/mattermost/mattermost-server/v5/app"
	"github.com/mattermost/mattermost-server/v5/model"
)

type MeProvider struct {
}

const (
	CMD_ME = "me"
)

func init() {
	app.RegisterCommandProvider(&MeProvider{})
}

func (*MeProvider) GetTrigger() string {
	return CMD_ME
}

func (*MeProvider) GetCommand(a *app.App, T goi18n.TranslateFunc) *model.Command {
	return &model.Command{
		Trigger:          CMD_ME,
		AutoComplete:     true,
		AutoCompleteDesc: T("api.command_me.desc"),
		AutoCompleteHint: T("api.command_me.hint"),
		DisplayName:      T("api.command_me.name"),
	}
}

func (*MeProvider) DoCommand(a *app.App, args *model.CommandArgs, message string) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
		Type:         model.POST_ME,
		Text:         "*" + message + "*",
		Props: model.StringInterface{
			"message": message,
		},
	}
}
