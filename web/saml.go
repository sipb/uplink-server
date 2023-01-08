// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package web

import (
	b64 "encoding/base64"
	"net/http"
	"strconv"
	"strings"

	"github.com/mattermost/mattermost-server/v5/audit"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/shared/mlog"
	"github.com/mattermost/mattermost-server/v5/utils"
)

func (w *Web) InitSaml() {
	w.MainRouter.Handle("/login/sso/saml", w.ApiHandler(loginWithSaml)).Methods("GET")
	w.MainRouter.Handle("/login/sso/saml", w.ApiHandlerTrustRequester(completeSaml)).Methods("POST")
}

func loginWithSaml(c *Context, w http.ResponseWriter, r *http.Request) {
	samlInterface := c.App.Saml()

	if samlInterface == nil {
		c.Err = model.NewAppError("loginWithSaml", "api.user.saml.not_available.app_error", nil, "", http.StatusFound)
		return
	}

	teamId, err := c.App.GetTeamIdFromQuery(r.URL.Query())
	if err != nil {
		c.Err = err
		return
	}
	action := r.URL.Query().Get("action")
	isMobile := action == model.OAUTH_ACTION_MOBILE
	redirectURL := r.URL.Query().Get("redirect_to")
	relayProps := map[string]string{}
	relayState := ""

	if action != "" {
		relayProps["team_id"] = teamId
		relayProps["action"] = action
		if action == model.OAUTH_ACTION_EMAIL_TO_SSO {
			relayProps["email"] = r.URL.Query().Get("email")
		}
	}

	if redirectURL != "" {
		if isMobile && !utils.IsValidMobileAuthRedirectURL(c.App.Config(), redirectURL) {
			invalidSchemeErr := model.NewAppError("loginWithOAuth", "api.invalid_custom_url_scheme", nil, "", http.StatusBadRequest)
			utils.RenderMobileError(c.App.Config(), w, invalidSchemeErr, redirectURL)
			return
		}
		relayProps["redirect_to"] = redirectURL
	}

	relayProps[model.USER_AUTH_SERVICE_IS_MOBILE] = strconv.FormatBool(isMobile)

	if len(relayProps) > 0 {
		relayState = b64.StdEncoding.EncodeToString([]byte(model.MapToJson(relayProps)))
	}

	data, err := samlInterface.BuildRequest(relayState)
	if err != nil {
		c.Err = err
		return
	}
	w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
	http.Redirect(w, r, data.URL, http.StatusFound)
}

func completeSaml(c *Context, w http.ResponseWriter, r *http.Request) {
	mlog.Debug("Touchstone login initiated")
	email := r.Header.Get("mail")
	kerb := strings.TrimSuffix(email, "@mit.edu")

	auditRec := c.MakeAuditRecord("TouchstoneLogin", audit.Fail)
	defer c.LogAuditRec(auditRec)
	c.LogAudit("attempt - login_id="+kerb)

	//Get user from kerb
	user, err := c.App.GetUserForLogin("", kerb)
	if err != nil {
		mlog.Error("Invalid kerb user!", mlog.String("kerb", kerb))
		c.LogAuditWithUserId(user.Id, "Touchstone login failure - kerb="+kerb)
		c.Err = err
		return
	}
	auditRec.AddMeta("obtained_user_id", user.Id)
	c.LogAuditWithUserId(user.Id, "obtained user")

	if err = c.App.CheckUserAllAuthenticationCriteria(user, ""); err != nil {
		mlog.Error("User authentication criteria check failed!")
		c.Err = err
		c.Err.StatusCode = http.StatusFound
		return
	}

	c.LogAuditWithUserId(user.Id, "authenticated")

	err = c.App.DoLogin(w, r, user, "", false, false, true)
	if err != nil {
		mlog.Error("Login Failed!")
		c.Err = err
		return
	}

	auditRec.Success()
	c.LogAuditWithUserId(user.Id, "success")

	c.App.AttachSessionCookies(w, r)
	http.Redirect(w, r, "https://uplink.mit.edu", http.StatusFound)
}
