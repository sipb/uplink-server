// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package localcachelayer

import (
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/store"
	"github.com/mattermost/mattermost-server/v5/store/storetest/mocks"
	"github.com/mattermost/mattermost-server/v5/testlib"
)

var mainHelper *testlib.MainHelper

func getMockStore() *mocks.Store {
	mockStore := mocks.Store{}

	fakeReaction := model.Reaction{PostId: "123"}
	mockReactionsStore := mocks.ReactionStore{}
	mockReactionsStore.On("Save", &fakeReaction).Return(&model.Reaction{}, nil)
	mockReactionsStore.On("Delete", &fakeReaction).Return(&model.Reaction{}, nil)
	mockReactionsStore.On("GetForPost", "123", false).Return([]*model.Reaction{&fakeReaction}, nil)
	mockReactionsStore.On("GetForPost", "123", true).Return([]*model.Reaction{&fakeReaction}, nil)
	mockStore.On("Reaction").Return(&mockReactionsStore)

	fakeRole := model.Role{Id: "123", Name: "role-name"}
	mockRolesStore := mocks.RoleStore{}
	mockRolesStore.On("Save", &fakeRole).Return(&model.Role{}, nil)
	mockRolesStore.On("Delete", "123").Return(&fakeRole, nil)
	mockRolesStore.On("GetByName", "role-name").Return(&fakeRole, nil)
	mockRolesStore.On("GetByNames", []string{"role-name"}).Return([]*model.Role{&fakeRole}, nil)
	mockRolesStore.On("PermanentDeleteAll").Return(nil)
	mockStore.On("Role").Return(&mockRolesStore)

	fakeScheme := model.Scheme{Id: "123", Name: "scheme-name"}
	mockSchemesStore := mocks.SchemeStore{}
	mockSchemesStore.On("Save", &fakeScheme).Return(&model.Scheme{}, nil)
	mockSchemesStore.On("Delete", "123").Return(&model.Scheme{}, nil)
	mockSchemesStore.On("Get", "123").Return(&fakeScheme, nil)
	mockSchemesStore.On("PermanentDeleteAll").Return(nil)
	mockStore.On("Scheme").Return(&mockSchemesStore)

	fakeWebhook := model.IncomingWebhook{Id: "123"}
	mockWebhookStore := mocks.WebhookStore{}
	mockWebhookStore.On("GetIncoming", "123", true).Return(&fakeWebhook, nil)
	mockWebhookStore.On("GetIncoming", "123", false).Return(&fakeWebhook, nil)
	mockStore.On("Webhook").Return(&mockWebhookStore)

	fakeEmoji := model.Emoji{Id: "123", Name: "name123"}
	mockEmojiStore := mocks.EmojiStore{}
	mockEmojiStore.On("Get", "123", true).Return(&fakeEmoji, nil)
	mockEmojiStore.On("Get", "123", false).Return(&fakeEmoji, nil)
	mockEmojiStore.On("GetByName", "name123", true).Return(&fakeEmoji, nil)
	mockEmojiStore.On("GetByName", "name123", false).Return(&fakeEmoji, nil)
	mockEmojiStore.On("Delete", &fakeEmoji, int64(0)).Return(nil)
	mockStore.On("Emoji").Return(&mockEmojiStore)

	mockCount := int64(10)
	mockGuestCount := int64(12)
	mockChannelStore := mocks.ChannelStore{}
	mockChannelStore.On("ClearCaches").Return()
	mockChannelStore.On("GetMemberCount", "id", true).Return(mockCount, nil)
	mockChannelStore.On("GetMemberCount", "id", false).Return(mockCount, nil)
	mockChannelStore.On("GetGuestCount", "id", true).Return(mockGuestCount, nil)
	mockChannelStore.On("GetGuestCount", "id", false).Return(mockGuestCount, nil)
	mockStore.On("Channel").Return(&mockChannelStore)

	mockPinnedPostsCount := int64(10)
	mockChannelStore.On("GetPinnedPostCount", "id", true).Return(mockPinnedPostsCount, nil)
	mockChannelStore.On("GetPinnedPostCount", "id", false).Return(mockPinnedPostsCount, nil)

	fakePosts := &model.PostList{}
	fakeOptions := model.GetPostsOptions{ChannelId: "123", PerPage: 30}
	mockPostStore := mocks.PostStore{}
	mockPostStore.On("GetPosts", fakeOptions, true).Return(fakePosts, nil)
	mockPostStore.On("GetPosts", fakeOptions, false).Return(fakePosts, nil)
	mockPostStore.On("InvalidateLastPostTimeCache", "12360")

	mockPostStoreOptions := model.GetPostsSinceOptions{
		ChannelId:        "channelId",
		Time:             1,
		SkipFetchThreads: false,
	}

	mockPostStoreEtagResult := fmt.Sprintf("%v.%v", model.CurrentVersion, 1)
	mockPostStore.On("ClearCaches")
	mockPostStore.On("InvalidateLastPostTimeCache", "channelId")
	mockPostStore.On("GetEtag", "channelId", true).Return(mockPostStoreEtagResult)
	mockPostStore.On("GetEtag", "channelId", false).Return(mockPostStoreEtagResult)
	mockPostStore.On("GetPostsSince", mockPostStoreOptions, true).Return(model.NewPostList(), nil)
	mockPostStore.On("GetPostsSince", mockPostStoreOptions, false).Return(model.NewPostList(), nil)
	mockStore.On("Post").Return(&mockPostStore)

	fakeUser := []*model.User{{Id: "123"}}
	mockUserStore := mocks.UserStore{}
	mockUserStore.On("GetProfileByIds", []string{"123"}, &store.UserGetByIdsOpts{}, true).Return(fakeUser, nil)
	mockUserStore.On("GetProfileByIds", []string{"123"}, &store.UserGetByIdsOpts{}, false).Return(fakeUser, nil)
	mockStore.On("User").Return(&mockUserStore)

	fakeUserTeamIds := []string{"1", "2", "3"}
	mockTeamStore := mocks.TeamStore{}
	mockTeamStore.On("GetUserTeamIds", "123", true).Return(fakeUserTeamIds, nil)
	mockTeamStore.On("GetUserTeamIds", "123", false).Return(fakeUserTeamIds, nil)
	mockStore.On("Team").Return(&mockTeamStore)

	return &mockStore
}

func TestMain(m *testing.M) {
	mainHelper = testlib.NewMainHelperWithOptions(nil)
	defer mainHelper.Close()

	initStores()
	mainHelper.Main(m)
	tearDownStores()
}
