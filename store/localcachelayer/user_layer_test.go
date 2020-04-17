// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package localcachelayer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/store"
	"github.com/mattermost/mattermost-server/v5/store/storetest"
	"github.com/mattermost/mattermost-server/v5/store/storetest/mocks"
)

func TestUserStore(t *testing.T) {
	StoreTestWithSqlSupplier(t, storetest.TestUserStore)
}

func TestUserStoreGetProfileByIdsCache(t *testing.T) {
	fakeUserIds := []string{"123"}
	fakeUser := []*model.User{{Id: "123"}}

	t.Run("first call not cached, second cached and returning same data", func(t *testing.T) {
		mockStore := getMockStore()
		mockCacheProvider := getMockCacheProvider()
		cachedStore := NewLocalCacheLayer(mockStore, nil, nil, mockCacheProvider)

		gotUser, err := cachedStore.User().GetProfileByIds(fakeUserIds, &store.UserGetByIdsOpts{}, true)
		require.Nil(t, err)
		assert.Equal(t, fakeUser, gotUser)
		mockStore.User().(*mocks.UserStore).AssertNumberOfCalls(t, "GetProfileByIds", 1)

		_, _ = cachedStore.User().GetProfileByIds(fakeUserIds, &store.UserGetByIdsOpts{}, true)
		mockStore.User().(*mocks.UserStore).AssertNumberOfCalls(t, "GetProfileByIds", 1)
	})

	t.Run("first call not cached, second force not cached", func(t *testing.T) {
		mockStore := getMockStore()
		mockCacheProvider := getMockCacheProvider()
		cachedStore := NewLocalCacheLayer(mockStore, nil, nil, mockCacheProvider)

		gotUser, err := cachedStore.User().GetProfileByIds(fakeUserIds, &store.UserGetByIdsOpts{}, true)
		require.Nil(t, err)
		assert.Equal(t, fakeUser, gotUser)
		mockStore.User().(*mocks.UserStore).AssertNumberOfCalls(t, "GetProfileByIds", 1)

		_, _ = cachedStore.User().GetProfileByIds(fakeUserIds, &store.UserGetByIdsOpts{}, false)
		mockStore.User().(*mocks.UserStore).AssertNumberOfCalls(t, "GetProfileByIds", 2)
	})

	t.Run("first call not cached, invalidate, and then not cached again", func(t *testing.T) {
		mockStore := getMockStore()
		mockCacheProvider := getMockCacheProvider()
		cachedStore := NewLocalCacheLayer(mockStore, nil, nil, mockCacheProvider)

		gotUser, err := cachedStore.User().GetProfileByIds(fakeUserIds, &store.UserGetByIdsOpts{}, true)
		require.Nil(t, err)
		assert.Equal(t, fakeUser, gotUser)
		mockStore.User().(*mocks.UserStore).AssertNumberOfCalls(t, "GetProfileByIds", 1)

		cachedStore.User().InvalidateProfileCacheForUser("123")

		_, _ = cachedStore.User().GetProfileByIds(fakeUserIds, &store.UserGetByIdsOpts{}, true)
		mockStore.User().(*mocks.UserStore).AssertNumberOfCalls(t, "GetProfileByIds", 2)
	})
}

func TestUserStoreProfilesInChannelCache(t *testing.T) {
	fakeChannelId := "123"
	fakeUserId := "456"
	fakeMap := map[string]*model.User{
		fakeUserId: {Id: "456"},
	}

	t.Run("first call not cached, second cached and returning same data", func(t *testing.T) {
		mockStore := getMockStore()
		mockCacheProvider := getMockCacheProvider()
		cachedStore := NewLocalCacheLayer(mockStore, nil, nil, mockCacheProvider)

		gotMap, err := cachedStore.User().GetAllProfilesInChannel(fakeChannelId, true)
		require.Nil(t, err)
		assert.Equal(t, fakeMap, gotMap)
		mockStore.User().(*mocks.UserStore).AssertNumberOfCalls(t, "GetAllProfilesInChannel", 1)

		_, _ = cachedStore.User().GetAllProfilesInChannel(fakeChannelId, true)
		mockStore.User().(*mocks.UserStore).AssertNumberOfCalls(t, "GetAllProfilesInChannel", 1)
	})

	t.Run("first call not cached, second force not cached", func(t *testing.T) {
		mockStore := getMockStore()
		mockCacheProvider := getMockCacheProvider()
		cachedStore := NewLocalCacheLayer(mockStore, nil, nil, mockCacheProvider)

		gotMap, err := cachedStore.User().GetAllProfilesInChannel(fakeChannelId, true)
		require.Nil(t, err)
		assert.Equal(t, fakeMap, gotMap)
		mockStore.User().(*mocks.UserStore).AssertNumberOfCalls(t, "GetAllProfilesInChannel", 1)

		_, _ = cachedStore.User().GetAllProfilesInChannel(fakeChannelId, false)
		mockStore.User().(*mocks.UserStore).AssertNumberOfCalls(t, "GetAllProfilesInChannel", 2)
	})

	t.Run("first call not cached, invalidate by channel, and then not cached again", func(t *testing.T) {
		mockStore := getMockStore()
		mockCacheProvider := getMockCacheProvider()
		cachedStore := NewLocalCacheLayer(mockStore, nil, nil, mockCacheProvider)

		gotMap, err := cachedStore.User().GetAllProfilesInChannel(fakeChannelId, true)
		require.Nil(t, err)
		assert.Equal(t, fakeMap, gotMap)
		mockStore.User().(*mocks.UserStore).AssertNumberOfCalls(t, "GetAllProfilesInChannel", 1)

		cachedStore.User().InvalidateProfilesInChannelCache("123")

		_, _ = cachedStore.User().GetAllProfilesInChannel(fakeChannelId, true)
		mockStore.User().(*mocks.UserStore).AssertNumberOfCalls(t, "GetAllProfilesInChannel", 2)
	})

	t.Run("first call not cached, invalidate by user, and then not cached again", func(t *testing.T) {
		mockStore := getMockStore()
		mockCacheProvider := getMockCacheProvider()
		cachedStore := NewLocalCacheLayer(mockStore, nil, nil, mockCacheProvider)

		gotMap, err := cachedStore.User().GetAllProfilesInChannel(fakeChannelId, true)
		require.Nil(t, err)
		assert.Equal(t, fakeMap, gotMap)
		mockStore.User().(*mocks.UserStore).AssertNumberOfCalls(t, "GetAllProfilesInChannel", 1)

		cachedStore.User().InvalidateProfilesInChannelCacheByUser("456")

		_, _ = cachedStore.User().GetAllProfilesInChannel(fakeChannelId, true)
		mockStore.User().(*mocks.UserStore).AssertNumberOfCalls(t, "GetAllProfilesInChannel", 2)
	})
}
func TestUserStoreGetCache(t *testing.T) {
	fakeUserId := "123"
	fakeUser := &model.User{Id: "123"}
	t.Run("first call not cached, second cached and returning same data", func(t *testing.T) {
		mockStore := getMockStore()
		mockCacheProvider := getMockCacheProvider()
		cachedStore := NewLocalCacheLayer(mockStore, nil, nil, mockCacheProvider)

		gotUser, err := cachedStore.User().Get(fakeUserId)
		require.Nil(t, err)
		assert.Equal(t, fakeUser, gotUser)
		mockStore.User().(*mocks.UserStore).AssertNumberOfCalls(t, "Get", 1)

		_, _ = cachedStore.User().Get(fakeUserId)
		mockStore.User().(*mocks.UserStore).AssertNumberOfCalls(t, "Get", 1)
	})

	t.Run("first call not cached, invalidate, and then not cached again", func(t *testing.T) {
		mockStore := getMockStore()
		mockCacheProvider := getMockCacheProvider()
		cachedStore := NewLocalCacheLayer(mockStore, nil, nil, mockCacheProvider)

		gotUser, err := cachedStore.User().Get(fakeUserId)
		require.Nil(t, err)
		assert.Equal(t, fakeUser, gotUser)
		mockStore.User().(*mocks.UserStore).AssertNumberOfCalls(t, "Get", 1)

		cachedStore.User().InvalidateProfileCacheForUser("123")

		_, _ = cachedStore.User().Get(fakeUserId)
		mockStore.User().(*mocks.UserStore).AssertNumberOfCalls(t, "Get", 2)
	})
}
