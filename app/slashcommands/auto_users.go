// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package slashcommands

import (
	"errors"
	"net/http"

	"github.com/mattermost/mattermost-server/v5/app"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/store"
	"github.com/mattermost/mattermost-server/v5/utils"
)

type AutoUserCreator struct {
	app          *app.App
	client       *model.Client4
	team         *model.Team
	EmailLength  utils.Range
	EmailCharset string
	NameLength   utils.Range
	NameCharset  string
	Fuzzy        bool
}

func NewAutoUserCreator(a *app.App, client *model.Client4, team *model.Team) *AutoUserCreator {
	return &AutoUserCreator{
		app:          a,
		client:       client,
		team:         team,
		EmailLength:  USER_EMAIL_LEN,
		EmailCharset: utils.LOWERCASE,
		NameLength:   USER_NAME_LEN,
		NameCharset:  utils.LOWERCASE,
		Fuzzy:        false,
	}
}

// Basic test team and user so you always know one
func CreateBasicUser(a *app.App, client *model.Client4) *model.AppError {
	found, _ := client.TeamExists(BTEST_TEAM_NAME, "")
	if found {
		return nil
	}

	newteam := &model.Team{DisplayName: BTEST_TEAM_DISPLAY_NAME, Name: BTEST_TEAM_NAME, Email: BTEST_TEAM_EMAIL, Type: BTEST_TEAM_TYPE}
	basicteam, resp := client.CreateTeam(newteam)
	if resp.Error != nil {
		return resp.Error
	}
	newuser := &model.User{Email: BTEST_USER_EMAIL, Nickname: BTEST_USER_NAME, Password: BTEST_USER_PASSWORD}
	ruser, resp := client.CreateUser(newuser)
	if resp.Error != nil {
		return resp.Error
	}
	_, err := a.Srv().Store.User().VerifyEmail(ruser.Id, ruser.Email)
	if err != nil {
		return err
	}
	if _, nErr := a.Srv().Store.Team().SaveMember(&model.TeamMember{TeamId: basicteam.Id, UserId: ruser.Id}, *a.Config().TeamSettings.MaxUsersPerTeam); nErr != nil {
		var appErr *model.AppError
		var conflictErr *store.ErrConflict
		var limitExceededErr *store.ErrLimitExceeded
		switch {
		case errors.As(nErr, &appErr): // in case we haven't converted to plain error.
			return appErr
		case errors.As(nErr, &conflictErr):
			return model.NewAppError("CreateBasicUser", "app.create_basic_user.save_member.conflict.app_error", nil, nErr.Error(), http.StatusBadRequest)
		case errors.As(nErr, &limitExceededErr):
			return model.NewAppError("CreateBasicUser", "app.create_basic_user.save_member.max_accounts.app_error", nil, nErr.Error(), http.StatusBadRequest)
		default: // last fallback in case it doesn't map to an existing app error.
			return model.NewAppError("CreateBasicUser", "app.create_basic_user.save_member.app_error", nil, nErr.Error(), http.StatusInternalServerError)
		}
	}

	return nil
}

func (cfg *AutoUserCreator) createRandomUser() (*model.User, error) {
	var userEmail string
	var userName string
	if cfg.Fuzzy {
		userEmail = "success+" + model.NewId() + "@simulator.amazonses.com"
		userName = utils.FuzzName()
	} else {
		userEmail = "success+" + model.NewId() + "@simulator.amazonses.com"
		userName = utils.RandomName(cfg.NameLength, cfg.NameCharset)
	}

	user := &model.User{
		Email:    userEmail,
		Nickname: userName,
		Password: USER_PASSWORD}

	ruser, resp := cfg.client.CreateUserWithInviteId(user, cfg.team.InviteId)
	if resp.Error != nil {
		return nil, resp.Error
	}

	status := &model.Status{UserId: ruser.Id, Status: model.STATUS_ONLINE, Manual: false, LastActivityAt: model.GetMillis(), ActiveChannel: ""}
	if err := cfg.app.Srv().Store.Status().SaveOrUpdate(status); err != nil {
		return nil, err
	}

	// We need to cheat to verify the user's email
	_, err := cfg.app.Srv().Store.User().VerifyEmail(ruser.Id, ruser.Email)
	if err != nil {
		return nil, err
	}

	return ruser, nil
}

func (cfg *AutoUserCreator) CreateTestUsers(num utils.Range) ([]*model.User, error) {
	numUsers := utils.RandIntFromRange(num)
	users := make([]*model.User, numUsers)

	for i := 0; i < numUsers; i++ {
		var err error
		users[i], err = cfg.createRandomUser()
		if err != nil {
			return nil, err
		}
	}

	return users, nil
}
