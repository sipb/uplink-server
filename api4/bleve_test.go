// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api4

import (
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
)

func TestBlevePurgeIndexes(t *testing.T) {
	th := Setup(t)
	defer th.TearDown()

	t.Run("as system user", func(t *testing.T) {
		_, resp := th.Client.PurgeBleveIndexes()
		CheckForbiddenStatus(t, resp)
	})

	t.Run("as system user with write experimental permission", func(t *testing.T) {
		th.AddPermissionToRole(model.PERMISSION_PURGE_BLEVE_INDEXES.Id, model.SYSTEM_USER_ROLE_ID)
		defer th.RemovePermissionFromRole(model.PERMISSION_SYSCONSOLE_WRITE_EXPERIMENTAL.Id, model.SYSTEM_USER_ROLE_ID)
		_, resp := th.Client.PurgeBleveIndexes()
		CheckOKStatus(t, resp)
	})

	t.Run("as system admin", func(t *testing.T) {
		_, resp := th.SystemAdminClient.PurgeBleveIndexes()
		CheckOKStatus(t, resp)
	})

	t.Run("as restricted system admin", func(t *testing.T) {
		th.App.UpdateConfig(func(cfg *model.Config) { *cfg.ExperimentalSettings.RestrictSystemAdmin = true })

		_, resp := th.SystemAdminClient.PurgeBleveIndexes()
		CheckForbiddenStatus(t, resp)
	})
}
