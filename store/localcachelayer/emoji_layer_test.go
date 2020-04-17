// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package localcachelayer

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/store/storetest"
	"github.com/mattermost/mattermost-server/store/storetest/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmojiStore(t *testing.T) {
	StoreTest(t, storetest.TestEmojiStore)
}

func TestEmojiStoreCache(t *testing.T) {
	fakeEmoji := model.Emoji{Id: "123", Name: "name123"}

	t.Run("first call by id not cached, second cached and returning same data", func(t *testing.T) {
		mockStore := getMockStore()
		cachedStore := NewLocalCacheLayer(mockStore, nil, nil)

		emoji, err := cachedStore.Emoji().Get("123", true)
		require.Nil(t, err)
		assert.Equal(t, emoji, &fakeEmoji)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "Get", 1)
		emoji, err = cachedStore.Emoji().Get("123", true)
		require.Nil(t, err)
		assert.Equal(t, emoji, &fakeEmoji)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "Get", 1)
	})

	t.Run("first call by name not cached, second cached and returning same data", func(t *testing.T) {
		mockStore := getMockStore()
		cachedStore := NewLocalCacheLayer(mockStore, nil, nil)

		emoji, err := cachedStore.Emoji().GetByName("name123", true)
		require.Nil(t, err)
		assert.Equal(t, emoji, &fakeEmoji)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "GetByName", 1)
		emoji, err = cachedStore.Emoji().GetByName("name123", true)
		require.Nil(t, err)
		assert.Equal(t, emoji, &fakeEmoji)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "GetByName", 1)
	})

	t.Run("first call by id not cached, second force no cached", func(t *testing.T) {
		mockStore := getMockStore()
		cachedStore := NewLocalCacheLayer(mockStore, nil, nil)

		cachedStore.Emoji().Get("123", true)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "Get", 1)
		cachedStore.Emoji().Get("123", false)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "Get", 2)
	})

	t.Run("first call by name not cached, second force no cached", func(t *testing.T) {
		mockStore := getMockStore()
		cachedStore := NewLocalCacheLayer(mockStore, nil, nil)

		cachedStore.Emoji().GetByName("name123", true)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "GetByName", 1)
		cachedStore.Emoji().GetByName("name123", false)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "GetByName", 2)
	})

	t.Run("first call by id force no cached, second not cached, third cached", func(t *testing.T) {
		mockStore := getMockStore()
		cachedStore := NewLocalCacheLayer(mockStore, nil, nil)

		cachedStore.Emoji().Get("123", false)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "Get", 1)
		cachedStore.Emoji().Get("123", true)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "Get", 2)
		cachedStore.Emoji().Get("123", true)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "Get", 2)
	})

	t.Run("first call by id force no cached, second not cached, third cached", func(t *testing.T) {
		mockStore := getMockStore()
		cachedStore := NewLocalCacheLayer(mockStore, nil, nil)

		cachedStore.Emoji().GetByName("name123", false)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "GetByName", 1)
		cachedStore.Emoji().GetByName("name123", true)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "GetByName", 2)
		cachedStore.Emoji().GetByName("name123", true)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "GetByName", 2)
	})

	t.Run("first call by id, second call by name cached", func(t *testing.T) {
		mockStore := getMockStore()
		cachedStore := NewLocalCacheLayer(mockStore, nil, nil)

		cachedStore.Emoji().Get("123", true)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "Get", 1)
		cachedStore.Emoji().GetByName("name123", true)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "GetByName", 0)
	})

	t.Run("first call by name, second call by id cached", func(t *testing.T) {
		mockStore := getMockStore()
		cachedStore := NewLocalCacheLayer(mockStore, nil, nil)

		cachedStore.Emoji().GetByName("name123", true)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "GetByName", 1)
		cachedStore.Emoji().Get("123", true)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "Get", 0)
	})

	t.Run("first call by id not cached, invalidate, and then not cached again", func(t *testing.T) {
		mockStore := getMockStore()
		cachedStore := NewLocalCacheLayer(mockStore, nil, nil)

		cachedStore.Emoji().Get("123", true)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "Get", 1)
		cachedStore.Emoji().Delete(&fakeEmoji, 0)
		cachedStore.Emoji().Get("123", true)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "Get", 2)
	})

	t.Run("first call by name not cached, invalidate, and then not cached again", func(t *testing.T) {
		mockStore := getMockStore()
		cachedStore := NewLocalCacheLayer(mockStore, nil, nil)

		cachedStore.Emoji().GetByName("name123", true)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "GetByName", 1)
		cachedStore.Emoji().Delete(&fakeEmoji, 0)
		cachedStore.Emoji().GetByName("name123", true)
		mockStore.Emoji().(*mocks.EmojiStore).AssertNumberOfCalls(t, "GetByName", 2)
	})
}
