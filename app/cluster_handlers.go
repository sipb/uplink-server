// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"strings"

	"github.com/mattermost/mattermost-server/v5/model"
)

// RegisterAllClusterMessageHandlers registers the cluster message handlers that are handled by the App layer.
//
// The cluster event handlers are spread across this function and
// NewLocalCacheLayer. Be careful to not have duplicated handlers here and
// there.
func (a *App) RegisterAllClusterMessageHandlers() {
	a.Cluster.RegisterClusterMessageHandler(model.CLUSTER_EVENT_PUBLISH, a.ClusterPublishHandler)
	a.Cluster.RegisterClusterMessageHandler(model.CLUSTER_EVENT_UPDATE_STATUS, a.ClusterUpdateStatusHandler)
	a.Cluster.RegisterClusterMessageHandler(model.CLUSTER_EVENT_INVALIDATE_ALL_CACHES, a.ClusterInvalidateAllCachesHandler)
	a.Cluster.RegisterClusterMessageHandler(model.CLUSTER_EVENT_INVALIDATE_CACHE_FOR_CHANNEL_MEMBERS_NOTIFY_PROPS, a.ClusterInvalidateCacheForChannelMembersNotifyPropHandler)
	a.Cluster.RegisterClusterMessageHandler(model.CLUSTER_EVENT_INVALIDATE_CACHE_FOR_CHANNEL_BY_NAME, a.ClusterInvalidateCacheForChannelByNameHandler)
	a.Cluster.RegisterClusterMessageHandler(model.CLUSTER_EVENT_INVALIDATE_CACHE_FOR_USER, a.ClusterInvalidateCacheForUserHandler)
	a.Cluster.RegisterClusterMessageHandler(model.CLUSTER_EVENT_INVALIDATE_CACHE_FOR_USER_TEAMS, a.ClusterInvalidateCacheForUserTeamsHandler)
	a.Cluster.RegisterClusterMessageHandler(model.CLUSTER_EVENT_CLEAR_SESSION_CACHE_FOR_USER, a.ClusterClearSessionCacheForUserHandler)
	a.Cluster.RegisterClusterMessageHandler(model.CLUSTER_EVENT_CLEAR_SESSION_CACHE_FOR_ALL_USERS, a.ClusterClearSessionCacheForAllUsersHandler)
	a.Cluster.RegisterClusterMessageHandler(model.CLUSTER_EVENT_INSTALL_PLUGIN, a.ClusterInstallPluginHandler)
	a.Cluster.RegisterClusterMessageHandler(model.CLUSTER_EVENT_REMOVE_PLUGIN, a.ClusterRemovePluginHandler)
	a.Cluster.RegisterClusterMessageHandler(model.CLUSTER_EVENT_BUSY_STATE_CHANGED, a.ClusterBusyStateChgHandler)
}

func (a *App) ClusterPublishHandler(msg *model.ClusterMessage) {
	event := model.WebSocketEventFromJson(strings.NewReader(msg.Data))
	a.PublishSkipClusterSend(event)
}

func (a *App) ClusterUpdateStatusHandler(msg *model.ClusterMessage) {
	status := model.StatusFromJson(strings.NewReader(msg.Data))
	a.AddStatusCacheSkipClusterSend(status)
}

func (a *App) ClusterInvalidateAllCachesHandler(msg *model.ClusterMessage) {
	a.InvalidateAllCachesSkipSend()
}

func (a *App) ClusterInvalidateCacheForChannelMembersNotifyPropHandler(msg *model.ClusterMessage) {
	a.InvalidateCacheForChannelMembersNotifyPropsSkipClusterSend(msg.Data)
}

func (a *App) ClusterInvalidateCacheForChannelByNameHandler(msg *model.ClusterMessage) {
	a.InvalidateCacheForChannelByNameSkipClusterSend(msg.Props["id"], msg.Props["name"])
}

func (a *App) ClusterInvalidateCacheForUserHandler(msg *model.ClusterMessage) {
	a.InvalidateCacheForUserSkipClusterSend(msg.Data)
}

func (a *App) ClusterInvalidateCacheForUserTeamsHandler(msg *model.ClusterMessage) {
	a.InvalidateCacheForUserTeamsSkipClusterSend(msg.Data)
}

func (a *App) ClusterClearSessionCacheForUserHandler(msg *model.ClusterMessage) {
	a.ClearSessionCacheForUserSkipClusterSend(msg.Data)
}

func (a *App) ClusterClearSessionCacheForAllUsersHandler(msg *model.ClusterMessage) {
	a.ClearSessionCacheForAllUsersSkipClusterSend()
}

func (a *App) ClusterInstallPluginHandler(msg *model.ClusterMessage) {
	a.InstallPluginFromData(model.PluginEventDataFromJson(strings.NewReader(msg.Data)))
}

func (a *App) ClusterRemovePluginHandler(msg *model.ClusterMessage) {
	a.RemovePluginFromData(model.PluginEventDataFromJson(strings.NewReader(msg.Data)))
}

func (a *App) ClusterBusyStateChgHandler(msg *model.ClusterMessage) {
	a.ServerBusyStateChanged(model.ServerBusyStateFromJson(strings.NewReader(msg.Data)))
}
