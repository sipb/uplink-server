// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"math/rand"
	"time"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/utils"
)

type TestEnvironment struct {
	Teams        []*model.Team
	Environments []TeamEnvironment
}

func CreateTestEnvironmentWithTeams(a *App, client *model.Client4, rangeTeams utils.Range, rangeChannels utils.Range, rangeUsers utils.Range, rangePosts utils.Range, fuzzy bool) (TestEnvironment, error) {
	rand.Seed(time.Now().UTC().UnixNano())

	teamCreator := NewAutoTeamCreator(client)
	teamCreator.Fuzzy = fuzzy
	teams, err := teamCreator.CreateTestTeams(rangeTeams)
	if err != nil {
		return TestEnvironment{}, err
	}

	environment := TestEnvironment{teams, make([]TeamEnvironment, len(teams))}

	for i, team := range teams {
		userCreator := NewAutoUserCreator(a, client, team)
		userCreator.Fuzzy = fuzzy
		randomUser, err := userCreator.createRandomUser()
		if err != nil {
			return TestEnvironment{}, err
		}
		client.LoginById(randomUser.Id, USER_PASSWORD)
		teamEnvironment, err := CreateTestEnvironmentInTeam(a, client, team, rangeChannels, rangeUsers, rangePosts, fuzzy)
		if err != nil {
			return TestEnvironment{}, err
		}
		environment.Environments[i] = teamEnvironment
	}

	return environment, nil
}

func CreateTestEnvironmentInTeam(a *App, client *model.Client4, team *model.Team, rangeChannels utils.Range, rangeUsers utils.Range, rangePosts utils.Range, fuzzy bool) (TeamEnvironment, error) {
	rand.Seed(time.Now().UTC().UnixNano())

	// We need to create at least one user
	if rangeUsers.Begin <= 0 {
		rangeUsers.Begin = 1
	}

	userCreator := NewAutoUserCreator(a, client, team)
	userCreator.Fuzzy = fuzzy
	users, err := userCreator.CreateTestUsers(rangeUsers)
	if err != nil {
		return TeamEnvironment{}, nil
	}
	usernames := make([]string, len(users))
	for i, user := range users {
		usernames[i] = user.Username
	}

	channelCreator := NewAutoChannelCreator(client, team)
	channelCreator.Fuzzy = fuzzy
	channels, err := channelCreator.CreateTestChannels(rangeChannels)
	if err != nil {
		return TeamEnvironment{}, nil
	}

	// Have every user join every channel
	for _, user := range users {
		for _, channel := range channels {
			_, resp := client.LoginById(user.Id, USER_PASSWORD)
			if resp.Error != nil {
				return TeamEnvironment{}, resp.Error
			}

			_, resp = client.AddChannelMember(channel.Id, user.Id)
			if resp.Error != nil {
				return TeamEnvironment{}, resp.Error
			}
		}
	}

	numPosts := utils.RandIntFromRange(rangePosts)
	numImages := utils.RandIntFromRange(rangePosts) / 4
	for j := 0; j < numPosts; j++ {
		user := users[utils.RandIntFromRange(utils.Range{Begin: 0, End: len(users) - 1})]
		_, resp := client.LoginById(user.Id, USER_PASSWORD)
		if resp.Error != nil {
			return TeamEnvironment{}, resp.Error
		}

		for i, channel := range channels {
			postCreator := NewAutoPostCreator(client, channel.Id)
			postCreator.HasImage = i < numImages
			postCreator.Users = usernames
			postCreator.Fuzzy = fuzzy
			_, err := postCreator.CreateRandomPost()
			if err != nil {
				return TeamEnvironment{}, err
			}
		}
	}

	return TeamEnvironment{users, channels}, nil
}
