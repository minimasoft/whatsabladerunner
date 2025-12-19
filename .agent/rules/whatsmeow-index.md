---
trigger: always_on
---

func DecryptMediaRetryNotification(evt *events.MediaRetry, mediaKey []byte) (*waMmsRetry.MediaRetryNotification, error)
func GenerateFacebookMessageID() int64
func GenerateMessageID() types.MessageIDdeprecated
func GetLatestVersion(ctx context.Context, httpClient *http.Client) (*store.WAVersionContainer, error)
func HashPollOptions(optionNames []string) [][]byte
func ParseDisappearingTimerString(val string) (time.Duration, bool)
type APNsPushConfig
func (apc *APNsPushConfig) GetPushConfigAttrs() waBinary.Attrs
type Client
func NewClient(deviceStore *store.Device, log waLog.Logger) *Client
func (cli *Client) AcceptTOSNotice(ctx context.Context, noticeID, stage string) error
func (cli *Client) AddEventHandler(handler EventHandler) uint32
func (cli *Client) AddEventHandlerWithSuccessStatus(handler EventHandlerWithSuccessStatus) uint32
func (cli *Client) BuildEdit(chat types.JID, id types.MessageID, newContent *waE2E.Message) *waE2E.Message
func (cli *Client) BuildHistorySyncRequest(lastKnownMessageInfo *types.MessageInfo, count int) *waE2E.Message
func (cli *Client) BuildMessageKey(chat, sender types.JID, id types.MessageID) *waCommon.MessageKey
func (cli *Client) BuildPollCreation(name string, optionNames []string, selectableOptionCount int) *waE2E.Message
func (cli *Client) BuildPollVote(ctx context.Context, pollInfo *types.MessageInfo, optionNames []string) (*waE2E.Message, error)
func (cli *Client) BuildReaction(chat, sender types.JID, id types.MessageID, reaction string) *waE2E.Message
func (cli *Client) BuildRevoke(chat, sender types.JID, id types.MessageID) *waE2E.Message
func (cli *Client) BuildUnavailableMessageRequest(chat, sender types.JID, id string) *waE2E.Message
func (cli *Client) Connect() error
func (cli *Client) ConnectContext(ctx context.Context) error
func (cli *Client) CreateGroup(ctx context.Context, req ReqCreateGroup) (*types.GroupInfo, error)
func (cli *Client) CreateNewsletter(ctx context.Context, params CreateNewsletterParams) (*types.NewsletterMetadata, error)
func (cli *Client) DangerousInternals() *DangerousInternalClientdeprecated
func (cli *Client) DecryptComment(ctx context.Context, comment *events.Message) (*waE2E.Message, error)
func (cli *Client) DecryptPollVote(ctx context.Context, vote *events.Message) (*waE2E.PollVoteMessage, error)
func (cli *Client) DecryptReaction(ctx context.Context, reaction *events.Message) (*waE2E.ReactionMessage, error)
func (cli *Client) DecryptSecretEncryptedMessage(ctx context.Context, evt *events.Message) (*waE2E.Message, error)
func (cli *Client) Disconnect()
func (cli *Client) Download(ctx context.Context, msg DownloadableMessage) ([]byte, error)
func (cli *Client) DownloadAny(ctx context.Context, msg *waE2E.Message) (data []byte, err error)deprecated
func (cli *Client) DownloadFB(ctx context.Context, transport *waMediaTransport.WAMediaTransport_Integral, ...) ([]byte, error)
func (cli *Client) DownloadFBToFile(ctx context.Context, transport *waMediaTransport.WAMediaTransport_Integral, ...) error
func (cli *Client) DownloadHistorySync(ctx context.Context, notif *waE2E.HistorySyncNotification, ...) (*waHistorySync.HistorySync, error)
func (cli *Client) DownloadMediaWithPath(ctx context.Context, directPath string, encFileHash, fileHash, mediaKey []byte, ...) (data []byte, err error)
func (cli *Client) DownloadMediaWithPathToFile(ctx context.Context, directPath string, encFileHash, fileHash, mediaKey []byte, ...) error
func (cli *Client) DownloadThumbnail(ctx context.Context, msg DownloadableThumbnail) ([]byte, error)
func (cli *Client) DownloadToFile(ctx context.Context, msg DownloadableMessage, file File) error
func (cli *Client) EncryptComment(ctx context.Context, rootMsgInfo *types.MessageInfo, comment *waE2E.Message) (*waE2E.Message, error)
func (cli *Client) EncryptPollVote(ctx context.Context, pollInfo *types.MessageInfo, vote *waE2E.PollVoteMessage) (*waE2E.PollUpdateMessage, error)
func (cli *Client) EncryptReaction(ctx context.Context, rootMsgInfo *types.MessageInfo, ...) (*waE2E.EncReactionMessage, error)
func (cli *Client) FetchAppState(ctx context.Context, name appstate.WAPatchName, fullSync, onlyIfNotSynced bool) error
func (cli *Client) FollowNewsletter(ctx context.Context, jid types.JID) error
func (cli *Client) GenerateMessageID() types.MessageID
func (cli *Client) GetBlocklist(ctx context.Context) (*types.Blocklist, error)
func (cli *Client) GetBotListV2(ctx context.Context) ([]types.BotListInfo, error)
func (cli *Client) GetBotProfiles(ctx context.Context, botInfo []types.BotListInfo) ([]types.BotProfileInfo, error)
func (cli *Client) GetBusinessProfile(ctx context.Context, jid types.JID) (*types.BusinessProfile, error)
func (cli *Client) GetContactQRLink(ctx context.Context, revoke bool) (string, error)
func (cli *Client) GetGroupInfo(ctx context.Context, jid types.JID) (*types.GroupInfo, error)
func (cli *Client) GetGroupInfoFromInvite(ctx context.Context, jid, inviter types.JID, code string, expiration int64) (*types.GroupInfo, error)
func (cli *Client) GetGroupInfoFromLink(ctx context.Context, code string) (*types.GroupInfo, error)
func (cli *Client) GetGroupInviteLink(ctx context.Context, jid types.JID, reset bool) (string, error)
func (cli *Client) GetGroupRequestParticipants(ctx context.Context, jid types.JID) ([]types.GroupParticipantRequest, error)
func (cli *Client) GetJoinedGroups(ctx context.Context) ([]*types.GroupInfo, error)
func (cli *Client) GetLinkedGroupsParticipants(ctx context.Context, community types.JID) ([]types.JID, error)
func (cli *Client) GetNewsletterInfo(ctx context.Context, jid types.JID) (*types.NewsletterMetadata, error)
func (cli *Client) GetNewsletterInfoWithInvite(ctx context.Context, key string) (*types.NewsletterMetadata, error)
func (cli *Client) GetNewsletterMessageUpdates(ctx context.Context, jid types.JID, params *GetNewsletterUpdatesParams) ([]*types.NewsletterMessage, error)
func (cli *Client) GetNewsletterMessages(ctx context.Context, jid types.JID, params *GetNewsletterMessagesParams) ([]*types.NewsletterMessage, error)
func (cli *Client) GetPrivacySettings(ctx context.Context) (settings types.PrivacySettings)
func (cli *Client) GetProfilePictureInfo(ctx context.Context, jid types.JID, params *GetProfilePictureParams) (*types.ProfilePictureInfo, error)
func (cli *Client) GetQRChannel(ctx context.Context) (<-chan QRChannelItem, error)
func (cli *Client) GetServerPushNotificationConfig(ctx context.Context) (*waBinary.Node, error)
func (cli *Client) GetStatusPrivacy(ctx context.Context) ([]types.StatusPrivacy, error)
func (cli *Client) GetSubGroups(ctx context.Context, community types.JID) ([]*types.GroupLinkTarget, error)
func (cli *Client) GetSubscribedNewsletters(ctx context.Context) ([]*types.NewsletterMetadata, error)
func (cli *Client) GetUserDevices(ctx context.Context, jids []types.JID) ([]types.JID, error)
func (cli *Client) GetUserDevicesContext(ctx context.Context, jids []types.JID) ([]types.JID, error)
func (cli *Client) GetUserInfo(ctx context.Context, jids []types.JID) (map[types.JID]types.UserInfo, error)
func (cli *Client) IsConnected() bool
func (cli *Client) IsLoggedIn() bool
func (cli *Client) IsOnWhatsApp(ctx context.Context, phones []string) ([]types.IsOnWhatsAppResponse, error)
func (cli *Client) JoinGroupWithInvite(ctx context.Context, jid, inviter types.JID, code string, expiration int64) error
func (cli *Client) JoinGroupWithLink(ctx context.Context, code string) (types.JID, error)
func (cli *Client) LeaveGroup(ctx context.Context, jid types.JID) error
func (cli *Client) LinkGroup(ctx context.Context, parent, child types.JID) error
func (cli *Client) Logout(ctx context.Context) error
func (cli *Client) MarkNotDirty(ctx context.Context, cleanType string, ts time.Time) error
func (cli *Client) MarkRead(ctx context.Context, ids []types.MessageID, timestamp time.Time, ...) error
func (cli *Client) NewsletterMarkViewed(ctx context.Context, jid types.JID, serverIDs []types.MessageServerID) error
func (cli *Client) NewsletterSendReaction(ctx context.Context, jid types.JID, serverID types.MessageServerID, ...) error
func (cli *Client) NewsletterSubscribeLiveUpdates(ctx context.Context, jid types.JID) (time.Duration, error)
func (cli *Client) NewsletterToggleMute(ctx context.Context, jid types.JID, mute bool) error
func (cli *Client) PairPhone(ctx context.Context, phone string, showPushNotification bool, ...) (string, error)
func (cli *Client) ParseWebMessage(chatJID types.JID, webMsg *waWeb.WebMessageInfo) (*events.Message, error)
func (cli *Client) RegisterForPushNotifications(ctx context.Context, pc PushConfig) error
func (cli *Client) RejectCall(ctx context.Context, callFrom types.JID, callID string) error
func (cli *Client) RemoveEventHandler(id uint32) bool
func (cli *Client) RemoveEventHandlers()
func (cli *Client) ResolveBusinessMessageLink(ctx context.Context, code string) (*types.BusinessMessageLinkTarget, error)
func (cli *Client) ResolveContactQRLink(ctx context.Context, code string) (*types.ContactQRLinkTarget, error)
func (cli *Client) RevokeMessage(ctx context.Context, chat types.JID, id types.MessageID) (SendResponse, error)deprecated
func (cli *Client) SendAppState(ctx context.Context, patch appstate.PatchInfo) error
func (cli *Client) SendChatPresence(ctx context.Context, jid types.JID, state types.ChatPresence, ...) error
func (cli *Client) SendFBMessage(ctx context.Context, to types.JID, message armadillo.RealMessageApplicationSub, ...) (resp SendResponse, err error)
func (cli *Client) SendMediaRetryReceipt(ctx context.Context, message *types.MessageInfo, mediaKey []byte) error
func (cli *Client) SendMessage(ctx context.Context, to types.JID, message *waE2E.Message, ...) (resp SendResponse, err error)
func (cli *Client) SendPresence(ctx context.Context, state types.Presence) error
func (cli *Client) SetDefaultDisappearingTimer(ctx context.Context, timer time.Duration) (err error)
func (cli *Client) SetDisappearingTimer(ctx context.Context, chat types.JID, timer time.Duration, settingTS time.Time) (err error)
func (cli *Client) SetForceActiveDeliveryReceipts(active bool)
func (cli *Client) SetGroupAnnounce(ctx context.Context, jid types.JID, announce bool) error
func (cli *Client) SetGroupDescription(ctx context.Context, jid types.JID, description string) error
func (cli *Client) SetGroupJoinApprovalMode(ctx context.Context, jid types.JID, mode bool) error
func (cli *Client) SetGroupLocked(ctx context.Context, jid types.JID, locked bool) error
func (cli *Client) SetGroupMemberAddMode(ctx context.Context, jid types.JID, mode types.GroupMemberAddMode) error
func (cli *Client) SetGroupName(ctx context.Context, jid types.JID, name string) error
func (cli *Client) SetGroupPhoto(ctx context.Context, jid types.JID, avatar []byte) (string, error)
func (cli *Client) SetGroupTopic(ctx context.Context, jid types.JID, previousID, newID, topic string) error
func (cli *Client) SetMediaHTTPClient(h *http.Client)
func (cli *Client) SetPassive(ctx context.Context, passive bool) error
func (cli *Client) SetPreLoginHTTPClient(h *http.Client)
func (cli *Client) SetPrivacySetting(ctx context.Context, name types.PrivacySettingType, value types.PrivacySetting) (settings types.PrivacySettings, err error)
func (cli *Client) SetProxy(proxy Proxy, opts ...SetProxyOptions)
func (cli *Client) SetProxyAddress(addr string, opts ...SetProxyOptions) error
func (cli *Client) SetSOCKSProxy(px proxy.Dialer, opts ...SetProxyOptions)
func (cli *Client) SetStatusMessage(ctx context.Context, msg string) error
func (cli *Client) SetWebsocketHTTPClient(h *http.Client)
func (cli *Client) StoreLIDPNMapping(ctx context.Context, first, second types.JID)
func (cli *Client) SubscribePresence(ctx context.Context, jid types.JID) error
func (cli *Client) TryFetchPrivacySettings(ctx context.Context, ignoreCache bool) (*types.PrivacySettings, error)
func (cli *Client) UnfollowNewsletter(ctx context.Context, jid types.JID) error
func (cli *Client) UnlinkGroup(ctx c