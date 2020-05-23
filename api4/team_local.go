// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api4

func (api *API) InitTeamLocal() {
	api.BaseRoutes.Teams.Handle("", api.ApiLocal(getAllTeams)).Methods("GET")
	api.BaseRoutes.TeamMembers.Handle("", api.ApiLocal(addTeamMember)).Methods("POST")
	api.BaseRoutes.TeamMember.Handle("", api.ApiLocal(removeTeamMember)).Methods("DELETE")
}
