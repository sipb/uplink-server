// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api4

import (
	"net/http"

	"github.com/mattermost/mattermost-server/v5/audit"
	"github.com/mattermost/mattermost-server/v5/model"
)

func (api *API) InitUserLocal() {
	api.BaseRoutes.Users.Handle("", api.ApiLocal(getUsers)).Methods("GET")
	api.BaseRoutes.Users.Handle("", api.ApiLocal(localPermanentDeleteAllUsers)).Methods("DELETE")
	api.BaseRoutes.Users.Handle("", api.ApiLocal(createUser)).Methods("POST")
	api.BaseRoutes.Users.Handle("/password/reset/send", api.ApiLocal(sendPasswordReset)).Methods("POST")
	api.BaseRoutes.Users.Handle("/ids", api.ApiLocal(getUsersByIds)).Methods("POST")

	api.BaseRoutes.User.Handle("", api.ApiLocal(getUser)).Methods("GET")
	api.BaseRoutes.User.Handle("", api.ApiLocal(updateUser)).Methods("PUT")
	api.BaseRoutes.User.Handle("/roles", api.ApiLocal(updateUserRoles)).Methods("PUT")
	api.BaseRoutes.User.Handle("/mfa", api.ApiLocal(updateUserMfa)).Methods("PUT")
	api.BaseRoutes.User.Handle("/active", api.ApiLocal(updateUserActive)).Methods("PUT")

	api.BaseRoutes.UserByUsername.Handle("", api.ApiLocal(localGetUserByUsername)).Methods("GET")
	api.BaseRoutes.UserByEmail.Handle("", api.ApiLocal(localGetUserByEmail)).Methods("GET")

	api.BaseRoutes.Users.Handle("/tokens/revoke", api.ApiLocal(revokeUserAccessToken)).Methods("POST")
	api.BaseRoutes.User.Handle("/tokens", api.ApiLocal(getUserAccessTokensForUser)).Methods("GET")
	api.BaseRoutes.User.Handle("/tokens", api.ApiLocal(createUserAccessToken)).Methods("POST")
}

func localPermanentDeleteAllUsers(c *Context, w http.ResponseWriter, r *http.Request) {
	auditRec := c.MakeAuditRecord("localPermanentDeleteAllUsers", audit.Fail)
	defer c.LogAuditRec(auditRec)

	if err := c.App.PermanentDeleteAllUsers(); err != nil {
		c.Err = err
		return
	}

	auditRec.Success()
	ReturnStatusOK(w)
}

func localGetUserByUsername(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUsername()
	if c.Err != nil {
		return
	}

	user, err := c.App.GetUserByUsername(c.Params.Username)
	if err != nil {
		return
	}

	etag := user.Etag(*c.App.Config().PrivacySettings.ShowFullName, *c.App.Config().PrivacySettings.ShowEmailAddress)

	if c.HandleEtag(etag, "Get User", w, r) {
		return
	}

	c.App.SanitizeProfile(user, c.IsSystemAdmin())
	w.Header().Set(model.HEADER_ETAG_SERVER, etag)
	w.Write([]byte(user.ToJson()))
}

func localGetUserByEmail(c *Context, w http.ResponseWriter, r *http.Request) {
	c.SanitizeEmail()
	if c.Err != nil {
		return
	}

	sanitizeOptions := c.App.GetSanitizeOptions(c.IsSystemAdmin())
	if !sanitizeOptions["email"] {
		c.Err = model.NewAppError("getUserByEmail", "api.user.get_user_by_email.permissions.app_error", nil, "userId="+c.App.Session().UserId, http.StatusForbidden)
		return
	}

	user, err := c.App.GetUserByEmail(c.Params.Email)
	if err != nil {
		c.Err = err
		return
	}

	etag := user.Etag(*c.App.Config().PrivacySettings.ShowFullName, *c.App.Config().PrivacySettings.ShowEmailAddress)

	if c.HandleEtag(etag, "Get User", w, r) {
		return
	}

	c.App.SanitizeProfile(user, c.IsSystemAdmin())
	w.Header().Set(model.HEADER_ETAG_SERVER, etag)
	w.Write([]byte(user.ToJson()))
}
