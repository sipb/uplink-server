// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/utils/fileutils"
)

func getHashedKey(key string) string {
	hash := sha256.New()
	hash.Write([]byte(key))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

func TestPluginKeyValueStore(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()

	pluginId := "testpluginid"

	defer func() {
		assert.Nil(t, th.App.DeletePluginKey(pluginId, "key"))
		assert.Nil(t, th.App.DeletePluginKey(pluginId, "key2"))
		assert.Nil(t, th.App.DeletePluginKey(pluginId, "key3"))
		assert.Nil(t, th.App.DeletePluginKey(pluginId, "key4"))
	}()

	assert.Nil(t, th.App.SetPluginKey(pluginId, "key", []byte("test")))
	ret, err := th.App.GetPluginKey(pluginId, "key")
	assert.Nil(t, err)
	assert.Equal(t, []byte("test"), ret)

	// Test inserting over existing entries
	assert.Nil(t, th.App.SetPluginKey(pluginId, "key", []byte("test2")))
	ret, err = th.App.GetPluginKey(pluginId, "key")
	assert.Nil(t, err)
	assert.Equal(t, []byte("test2"), ret)

	// Test getting non-existent key
	ret, err = th.App.GetPluginKey(pluginId, "notakey")
	assert.Nil(t, err)
	assert.Nil(t, ret)

	// Test deleting non-existent keys.
	assert.Nil(t, th.App.DeletePluginKey(pluginId, "notrealkey"))

	// Verify behaviour for the old approach that involved storing the hashed keys.
	hashedKey2 := getHashedKey("key2")
	kv := &model.PluginKeyValue{
		PluginId: pluginId,
		Key:      hashedKey2,
		Value:    []byte("test"),
		ExpireAt: 0,
	}

	_, err = th.App.Srv.Store.Plugin().SaveOrUpdate(kv)
	assert.Nil(t, err)

	// Test fetch by keyname (this key does not exist but hashed key will be used for lookup)
	ret, err = th.App.GetPluginKey(pluginId, "key2")
	assert.Nil(t, err)
	assert.Equal(t, kv.Value, ret)

	// Test fetch by hashed keyname
	ret, err = th.App.GetPluginKey(pluginId, hashedKey2)
	assert.Nil(t, err)
	assert.Equal(t, kv.Value, ret)

	// Test ListKeys
	assert.Nil(t, th.App.SetPluginKey(pluginId, "key3", []byte("test3")))
	assert.Nil(t, th.App.SetPluginKey(pluginId, "key4", []byte("test4")))

	list, err := th.App.ListPluginKeys(pluginId, 0, 1)
	assert.Nil(t, err)
	assert.Equal(t, []string{"key"}, list)

	list, err = th.App.ListPluginKeys(pluginId, 1, 1)
	assert.Nil(t, err)
	assert.Equal(t, []string{"key3"}, list)

	list, err = th.App.ListPluginKeys(pluginId, 0, 4)
	assert.Nil(t, err)
	assert.Equal(t, []string{"key", "key3", "key4", hashedKey2}, list)

	list, err = th.App.ListPluginKeys(pluginId, 0, 2)
	assert.Nil(t, err)
	assert.Equal(t, []string{"key", "key3"}, list)

	list, err = th.App.ListPluginKeys(pluginId, 1, 2)
	assert.Nil(t, err)
	assert.Equal(t, []string{"key4", hashedKey2}, list)

	list, err = th.App.ListPluginKeys(pluginId, 2, 2)
	assert.Nil(t, err)
	assert.Equal(t, []string{}, list)

	// List Keys bad input
	list, err = th.App.ListPluginKeys(pluginId, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, []string{"key", "key3", "key4", hashedKey2}, list)

	list, err = th.App.ListPluginKeys(pluginId, 0, -1)
	assert.Nil(t, err)
	assert.Equal(t, []string{"key", "key3", "key4", hashedKey2}, list)

	list, err = th.App.ListPluginKeys(pluginId, -1, 1)
	assert.Nil(t, err)
	assert.Equal(t, []string{"key"}, list)

	list, err = th.App.ListPluginKeys(pluginId, -1, 0)
	assert.Nil(t, err)
	assert.Equal(t, []string{"key", "key3", "key4", hashedKey2}, list)
}

func TestPluginKeyValueStoreCompareAndSet(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()

	pluginId := "testpluginid"

	defer func() {
		assert.Nil(t, th.App.DeletePluginKey(pluginId, "key"))
	}()

	// Set using Set api for key2
	assert.Nil(t, th.App.SetPluginKey(pluginId, "key2", []byte("test")))
	ret, err := th.App.GetPluginKey(pluginId, "key2")
	assert.Nil(t, err)
	assert.Equal(t, []byte("test"), ret)

	// Attempt to insert value for key2
	updated, err := th.App.CompareAndSetPluginKey(pluginId, "key2", nil, []byte("test2"))
	assert.Nil(t, err)
	assert.False(t, updated)
	ret, err = th.App.GetPluginKey(pluginId, "key2")
	assert.Nil(t, err)
	assert.Equal(t, []byte("test"), ret)

	// Insert new value for key
	updated, err = th.App.CompareAndSetPluginKey(pluginId, "key", nil, []byte("test"))
	assert.Nil(t, err)
	assert.True(t, updated)
	ret, err = th.App.GetPluginKey(pluginId, "key")
	assert.Nil(t, err)
	assert.Equal(t, []byte("test"), ret)

	// Should fail to insert again
	updated, err = th.App.CompareAndSetPluginKey(pluginId, "key", nil, []byte("test3"))
	assert.Nil(t, err)
	assert.False(t, updated)
	ret, err = th.App.GetPluginKey(pluginId, "key")
	assert.Nil(t, err)
	assert.Equal(t, []byte("test"), ret)

	// Test updating using incorrect old value
	updated, err = th.App.CompareAndSetPluginKey(pluginId, "key", []byte("oldvalue"), []byte("test3"))
	assert.Nil(t, err)
	assert.False(t, updated)
	ret, err = th.App.GetPluginKey(pluginId, "key")
	assert.Nil(t, err)
	assert.Equal(t, []byte("test"), ret)

	// Test updating using correct old value
	updated, err = th.App.CompareAndSetPluginKey(pluginId, "key", []byte("test"), []byte("test2"))
	assert.Nil(t, err)
	assert.True(t, updated)
	ret, err = th.App.GetPluginKey(pluginId, "key")
	assert.Nil(t, err)
	assert.Equal(t, []byte("test2"), ret)
}

func TestPluginKeyValueStoreSetWithOptionsJSON(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()

	pluginId := "testpluginid"

	defer func() {
		assert.Nil(t, th.App.DeletePluginKey(pluginId, "key"))
	}()

	t.Run("fails with a non-serializable object as the new value", func(t *testing.T) {
		result, err := th.App.SetPluginKeyWithOptions(pluginId, "key", func() {}, model.PluginKVSetOptions{
			EncodeJSON: true,
		})
		assert.False(t, result)
		assert.NotNil(t, err)

		// verify that after the failure it was not set
		ret, err := th.App.GetPluginKey(pluginId, "key")
		assert.Nil(t, err)
		assert.Equal(t, []byte(nil), ret)
	})

	t.Run("fails with a non-serializable object as the old value", func(t *testing.T) {
		result, err := th.App.SetPluginKeyWithOptions(pluginId, "key", map[string]interface{}{
			"val-a": 10,
		}, model.PluginKVSetOptions{
			EncodeJSON: true,
			Atomic:     true,
			OldValue:   func() {},
		})
		assert.False(t, result)
		assert.NotNil(t, err)

		// verify that after the failure it was not set
		ret, err := th.App.GetPluginKey(pluginId, "key")
		assert.Nil(t, err)
		assert.Equal(t, []byte(nil), ret)
	})

	t.Run("storing a value json encoded works", func(t *testing.T) {
		result, err := th.App.SetPluginKeyWithOptions(pluginId, "key", map[string]interface{}{
			"val-a": 10,
		}, model.PluginKVSetOptions{
			EncodeJSON: true,
		})
		assert.True(t, result)
		assert.Nil(t, err)

		// and I can get it back!
		ret, err := th.App.GetPluginKey(pluginId, "key")
		assert.Nil(t, err)
		assert.Equal(t, []byte(`{"val-a":10}`), ret)
	})

	t.Run("test that setting it atomic when it doesn't match doesn't change anything", func(t *testing.T) {
		result, err := th.App.SetPluginKeyWithOptions(pluginId, "key", map[string]interface{}{
			"val-a": 30,
		}, model.PluginKVSetOptions{
			EncodeJSON: true,
			Atomic:     true,
			OldValue: map[string]interface{}{
				"val-a": 20,
			},
		})
		assert.False(t, result)
		assert.Nil(t, err)

		// test that the value didn't change
		ret, err := th.App.GetPluginKey(pluginId, "key")
		assert.Nil(t, err)
		assert.Equal(t, []byte(`{"val-a":10}`), ret)
	})

	t.Run("test the atomic change with the proper old value", func(t *testing.T) {
		result, err := th.App.SetPluginKeyWithOptions(pluginId, "key", map[string]interface{}{
			"val-a": 30,
		}, model.PluginKVSetOptions{
			EncodeJSON: true,
			Atomic:     true,
			OldValue: map[string]interface{}{
				"val-a": 10,
			},
		})
		assert.True(t, result)
		assert.Nil(t, err)

		// test that the value did change
		ret, err := th.App.GetPluginKey(pluginId, "key")
		assert.Nil(t, err)
		assert.Equal(t, []byte(`{"val-a":30}`), ret)
	})
}

func TestPluginKeyValueStoreSetWithOptionsByteArray(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()

	pluginId := "testpluginid"

	defer func() {
		assert.Nil(t, th.App.DeletePluginKey(pluginId, "key"))
	}()

	// storing a value works
	result, err := th.App.SetPluginKeyWithOptions(pluginId, "key", []byte(`myvalue`), model.PluginKVSetOptions{})
	assert.True(t, result)
	assert.Nil(t, err)

	// and I can get it back!
	ret, err := th.App.GetPluginKey(pluginId, "key")
	assert.Nil(t, err)
	assert.Equal(t, []byte(`myvalue`), ret)

	// test that setting it atomic when it doesn't match doesn't change anything
	result, err = th.App.SetPluginKeyWithOptions(pluginId, "key", []byte(`newvalue`), model.PluginKVSetOptions{
		Atomic:   true,
		OldValue: []byte(`differentvalue`),
	})
	assert.False(t, result)
	assert.Nil(t, err)

	// test that the value didn't change
	ret, err = th.App.GetPluginKey(pluginId, "key")
	assert.Nil(t, err)
	assert.Equal(t, []byte(`myvalue`), ret)

	// now do the atomic change with the proper old value
	result, err = th.App.SetPluginKeyWithOptions(pluginId, "key", []byte(`newvalue`), model.PluginKVSetOptions{
		Atomic:   true,
		OldValue: []byte(`myvalue`),
	})
	assert.True(t, result)
	assert.Nil(t, err)

	// test that the value did change
	ret, err = th.App.GetPluginKey(pluginId, "key")
	assert.Nil(t, err)
	assert.Equal(t, []byte(`newvalue`), ret)
}

func TestServePluginRequest(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()

	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.PluginSettings.Enable = false })

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/plugins/foo/bar", nil)
	th.App.ServePluginRequest(w, r)
	assert.Equal(t, http.StatusNotImplemented, w.Result().StatusCode)
}

func TestPrivateServePluginRequest(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()

	testCases := []struct {
		Description string
		ConfigFunc  func(cfg *model.Config)
		URL         string
		ExpectedURL string
	}{
		{
			"no subpath",
			func(cfg *model.Config) {},
			"/plugins/id/endpoint",
			"/endpoint",
		},
		{
			"subpath",
			func(cfg *model.Config) { *cfg.ServiceSettings.SiteURL += "/subpath" },
			"/subpath/plugins/id/endpoint",
			"/endpoint",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Description, func(t *testing.T) {
			th.App.UpdateConfig(testCase.ConfigFunc)
			expectedBody := []byte("body")
			request := httptest.NewRequest(http.MethodGet, testCase.URL, bytes.NewReader(expectedBody))
			recorder := httptest.NewRecorder()

			handler := func(context *plugin.Context, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, testCase.ExpectedURL, r.URL.Path)

				body, _ := ioutil.ReadAll(r.Body)
				assert.Equal(t, expectedBody, body)
			}

			request = mux.SetURLVars(request, map[string]string{"plugin_id": "id"})

			th.App.servePluginRequest(recorder, request, handler)
		})
	}

}

func TestHandlePluginRequest(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()

	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.PluginSettings.Enable = false
		*cfg.ServiceSettings.EnableUserAccessTokens = true
	})

	token, err := th.App.CreateUserAccessToken(&model.UserAccessToken{
		UserId: th.BasicUser.Id,
	})
	require.Nil(t, err)

	var assertions func(*http.Request)
	router := mux.NewRouter()
	router.HandleFunc("/plugins/{plugin_id:[A-Za-z0-9\\_\\-\\.]+}/{anything:.*}", func(_ http.ResponseWriter, r *http.Request) {
		th.App.servePluginRequest(nil, r, func(_ *plugin.Context, _ http.ResponseWriter, r *http.Request) {
			assertions(r)
		})
	})

	r := httptest.NewRequest("GET", "/plugins/foo/bar", nil)
	r.Header.Add("Authorization", "Bearer "+token.Token)
	assertions = func(r *http.Request) {
		assert.Equal(t, "/bar", r.URL.Path)
		assert.Equal(t, th.BasicUser.Id, r.Header.Get("Mattermost-User-Id"))
	}
	router.ServeHTTP(nil, r)

	r = httptest.NewRequest("GET", "/plugins/foo/bar?a=b&access_token="+token.Token+"&c=d", nil)
	assertions = func(r *http.Request) {
		assert.Equal(t, "/bar", r.URL.Path)
		assert.Equal(t, "a=b&c=d", r.URL.RawQuery)
		assert.Equal(t, th.BasicUser.Id, r.Header.Get("Mattermost-User-Id"))
	}
	router.ServeHTTP(nil, r)

	r = httptest.NewRequest("GET", "/plugins/foo/bar?a=b&access_token=asdf&c=d", nil)
	assertions = func(r *http.Request) {
		assert.Equal(t, "/bar", r.URL.Path)
		assert.Equal(t, "a=b&c=d", r.URL.RawQuery)
		assert.Empty(t, r.Header.Get("Mattermost-User-Id"))
	}
	router.ServeHTTP(nil, r)
}

func TestGetPluginStatusesDisabled(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()

	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.PluginSettings.Enable = false
	})

	_, err := th.App.GetPluginStatuses()
	require.NotNil(t, err)
	require.EqualError(t, err, "GetPluginStatuses: Plugins have been disabled. Please check your logs for details., ")
}

func TestGetPluginStatuses(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()

	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.PluginSettings.Enable = true
	})

	pluginStatuses, err := th.App.GetPluginStatuses()
	require.Nil(t, err)
	require.NotNil(t, pluginStatuses)
}

func TestPluginSync(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()

	testCases := []struct {
		Description string
		ConfigFunc  func(cfg *model.Config)
	}{
		{
			"local",
			func(cfg *model.Config) {
				cfg.FileSettings.DriverName = model.NewString(model.IMAGE_DRIVER_LOCAL)
			},
		},
		{
			"s3",
			func(cfg *model.Config) {
				s3Host := os.Getenv("CI_MINIO_HOST")
				if s3Host == "" {
					s3Host = "localhost"
				}

				s3Port := os.Getenv("CI_MINIO_PORT")
				if s3Port == "" {
					s3Port = "9001"
				}

				s3Endpoint := fmt.Sprintf("%s:%s", s3Host, s3Port)
				cfg.FileSettings.DriverName = model.NewString(model.IMAGE_DRIVER_S3)
				cfg.FileSettings.AmazonS3AccessKeyId = model.NewString(model.MINIO_ACCESS_KEY)
				cfg.FileSettings.AmazonS3SecretAccessKey = model.NewString(model.MINIO_SECRET_KEY)
				cfg.FileSettings.AmazonS3Bucket = model.NewString(model.MINIO_BUCKET)
				cfg.FileSettings.AmazonS3Endpoint = model.NewString(s3Endpoint)
				cfg.FileSettings.AmazonS3Region = model.NewString("")
				cfg.FileSettings.AmazonS3SSL = model.NewBool(false)

			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Description, func(t *testing.T) {
			os.MkdirAll("./test-plugins", os.ModePerm)
			defer os.RemoveAll("./test-plugins")

			th.App.UpdateConfig(func(cfg *model.Config) {
				*cfg.PluginSettings.Enable = true
				*cfg.PluginSettings.Directory = "./test-plugins"
				*cfg.PluginSettings.ClientDirectory = "./test-client-plugins"
			})
			th.App.UpdateConfig(testCase.ConfigFunc)

			env, err := plugin.NewEnvironment(th.App.NewPluginAPI, "./test-plugins", "./test-client-plugins", th.App.Log)
			require.NoError(t, err)
			th.App.SetPluginsEnvironment(env)

			// New bundle in the file store case
			path, _ := fileutils.FindDir("tests")
			fileReader, err := os.Open(filepath.Join(path, "testplugin.tar.gz"))
			require.NoError(t, err)
			defer fileReader.Close()

			_, appErr := th.App.WriteFile(fileReader, th.App.getBundleStorePath("testplugin"))
			checkNoError(t, appErr)

			appErr = th.App.SyncPlugins()
			checkNoError(t, appErr)

			// Check if installed
			pluginStatus, err := env.Statuses()
			require.Nil(t, err)
			require.True(t, len(pluginStatus) == 1)
			require.Equal(t, pluginStatus[0].PluginId, "testplugin")

			// Bundle removed from the file store case
			appErr = th.App.RemoveFile(th.App.getBundleStorePath("testplugin"))
			checkNoError(t, appErr)

			appErr = th.App.SyncPlugins()
			checkNoError(t, appErr)

			// Check if removed
			pluginStatus, err = env.Statuses()
			require.Nil(t, err)
			require.True(t, len(pluginStatus) == 0)
		})
	}
}
