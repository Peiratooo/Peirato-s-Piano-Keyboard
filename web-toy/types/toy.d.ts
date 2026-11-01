/**
 * Bilibili Toy JS SDK global declarations, based on the developer-provided
 * Toy JS SDK declarations + capability list (2026-09-18).
 *
 * Loaded at runtime by `//s1.hdslb.com/bfs/seed/toy/app/sdk/toy-sdk.js`.
 * This is a type contract, NOT an SDK implementation or verification of
 * platform privileges. API limits can change; see docs/TOY_SDK_INTEGRATION.md.
 *
 * Window.toy is OPTIONAL here, intentionally: the app also runs in ordinary
 * browser development mode, or the SDK script may fail to load.
 * Do not treat toyOpenId as a login token or as proof of user identity.
 */
declare namespace ToySDK {
  type NavigateType = 'video' | 'space' | 'search' | 'opus' | 'tribee' | 'toy'
  interface NavigateReq {
    type: NavigateType
    id: string
    extra?: Record<string, string>
  }

  interface SaveImageReq {
    /** Prefer URL for large images. Base64 string is at most 5,242,880 chars. */
    url?: string
    base64Data?: string
    hintMsg?: string
  }
  interface SaveImageResp { localPath: string }
  interface ShareReq { /** Relative to current /toy/<slug>/; not an external URL. */ path: string }
  interface QrCodeReq { path?: string; /** Integer 80..1024; defaults to 320. */ size?: number }
  interface QrCodeResp { base64: string; url: string }

  type ToyDeviceType = 'phone' | 'tablet' | 'desktop' | 'unknown'
  type ToyOrientation = 'portrait' | 'landscape'
  interface ToyViewport { width: number; height: number }
  interface ToySafeArea { top: number; right: number; bottom: number; left: number }
  type ToyContainerField = 'deviceType'|'viewport'|'orientation'|'immersive'|'safeArea'
  interface ToyContainerState {
    deviceType: ToyDeviceType
    viewport: ToyViewport
    orientation: ToyOrientation
    immersive: boolean
    safeArea: ToySafeArea
    changedFields: ToyContainerField[]
  }
  interface SetContainerModeReq { orientation?: ToyOrientation|'auto'; immersive?: boolean }
  type ContainerStateListener = (state: ToyContainerState) => void

  interface UserProfileResp {
    avatar: string
    nickname: string
    /** Toy-local stable pseudonym, NOT a credential; may be omitted. */
    toyOpenId?: string
  }
  type ToyDataStatus =
    'ok'|'partial'|'unauthorized'|'denied'|'unsupported'|
    'toy_context_unavailable'|'author_mismatch'|'video_not_found'|
    'video_invisible'|'unavailable'|'invalid_argument'
  type ToyItemStatus = Exclude<ToyDataStatus, 'partial'>
  interface AuthorCertification { role: number; title: string; description: string; type: number }
  interface ChargingSummary { count: number; display?: { show: boolean; text?: string } }
  interface FanMedalConfig {
    name: string
    enabled: boolean
    levels: Array<{ level: number; minIntimacy: number; maxIntimacy?: number }>
  }
  interface AuthorProfile {
    nickname: string
    avatar: string
    sign: string
    certification?: AuthorCertification
    following: number
    follower: number
    archiveCount: number
    charging?: ChargingSummary
    fanMedal?: FanMedalConfig
    birthday?: number
  }
  interface AuthorProfileResp { status: ToyDataStatus; data?: AuthorProfile }

  type AuthorVideoRef = { aid: number; bvid?: never } | { bvid: string; aid?: never }
  interface AuthorVideosReq { videos: AuthorVideoRef[] }
  interface AuthorVideo {
    aid: number
    bvid: string
    title: string
    cover: string
    description: string
    publishTime: number
    duration: number
    partition?: string
    pages: Array<{ page: number; title: string; duration: number }>
    stat: {
      view: number; like: number; coin: number; favorite: number
      share: number; comment: number; danmaku: number
    }
    pay: { chargingPay: boolean; paid: boolean }
    seasonId?: number
    rank?: { now: number; highest: number }
  }
  interface AuthorVideoItem { ref: AuthorVideoRef; status: ToyItemStatus; data?: AuthorVideo }
  interface AuthorVideosResp { status: ToyDataStatus; items: AuthorVideoItem[] }
  interface AuthorRelation {
    isFollowing: boolean
    isAuthor: boolean
    isOldFan: boolean
    hasFanMedal: boolean
    fanMedalName?: string
    fanMedalLevel?: number
    isFanMedalActive?: boolean
    isCharging: boolean
    followTime?: number
  }
  interface AuthorRelationResp { status: ToyDataStatus; data?: AuthorRelation }
  type VideoUserActionsReq =
    { aids: number[]; videos?: never } |
    { videos: AuthorVideoRef[]; aids?: never }
  interface VideoUserActionItem {
    aid: number
    bvid: string
    ref?: AuthorVideoRef
    status: ToyItemStatus
    liked?: boolean
    coinCount?: number
    favorited?: boolean
  }
  interface VideoUserActionsResp { status: ToyDataStatus; items: VideoUserActionItem[] }

  type RankPeriod = 'all'|'month'|'week'|'day'
  /**
   * DISCREPANCY in supplied docs: original .d.ts says boards 1..3,
   * later capability list says 1..5. Type covers 1..5; deploy only board 1
   * until the live SDK confirms support for others.
   */
  type RankBoard = 1|2|3|4|5
  interface SubmitScoreReq {
    board?: RankBoard
    /** ABSOLUTE highest score, not a delta; integer -16777216..16777215. */
    score: number
  }
  interface SubmitScoreResp { score: number }
  interface RankListReq { board?: RankBoard; period?: RankPeriod; limit?: number }
  interface RankItem { rank: number; score: number; nickname: string; avatar: string }
  interface MyRankReq { board?: RankBoard; period?: RankPeriod }
  interface MyRankResp { ranked: boolean; rank: number; score: number }

  type InputSafetyStatus = 'ok'|'rejected'|'unavailable'
  interface CheckInputSafetyReq { text: string; scene?: string }
  interface CheckInputSafetyResp { status: InputSafetyStatus; passed: boolean }
  interface MediaRelayOptions { facingMode?: 'user'|'environment' }

  interface Toy {
    isSupport(ability: string): Promise<boolean>
    /** Requires direct user activation. */
    navigate(req: NavigateReq): Promise<void>
    /** Bilibili App only; web uses browser download. */
    saveImageToAlbum(req: SaveImageReq): Promise<SaveImageResp>
    /** Bilibili App only; path must remain inside same Toy. */
    share(req: ShareReq): Promise<void>
    getQrCode(req?: QrCodeReq): Promise<QrCodeResp>
    /** Bilibili App only. */
    closeBrowser(): Promise<void>
    /** May prompt for user consent; call after user interaction if no prior grant. */
    getUserProfile(): Promise<UserProfileResp>
    getAuthorProfile(): Promise<AuthorProfileResp>
    getAuthorVideos(req: AuthorVideosReq): Promise<AuthorVideosResp>
    getAuthorRelation(): Promise<AuthorRelationResp>
    getVideoUserActions(req: VideoUserActionsReq): Promise<VideoUserActionsResp>
    /** Logged-in user per-Toy storage; values <=1024 UTF-8 bytes. */
    getCloudStorage(keys?: string[]): Promise<Record<string, string>>
    setCloudStorage(items: Record<string, string>): Promise<void>
    removeCloudStorage(keys: string[]): Promise<void>
    /** Shared storage IDs are created & authorized on the Toy platform. */
    getSharedStorage(storageId: string, keys?: string[]): Promise<Record<string, string>>
    setSharedStorage(storageId: string, items: Record<string, string>): Promise<void>
    removeSharedStorage(storageId: string, keys: string[]): Promise<void>
    submitScore(req: SubmitScoreReq): Promise<SubmitScoreResp>
    getRankList(req?: RankListReq): Promise<RankItem[]>
    getMyRank(req?: MyRankReq): Promise<MyRankResp>
    checkInputSafety(req: CheckInputSafetyReq): Promise<CheckInputSafetyResp>
    /** Requires direct user activation, app/business & system permission. */
    requestCamera(options?: MediaRelayOptions): Promise<MediaStream>
    /** Requires direct user activation. */
    requestMicrophone(): Promise<MediaStream>
    /** MUST release the exact stream returned by requestCamera/Microphone. */
    stopMedia(stream: MediaStream): Promise<void>
    /** Bilibili App only; synchronous unsubscribe function. */
    onContainerChange(listener: ContainerStateListener): () => void
    /** Bilibili App only. */
    getContainerState(): Promise<ToyContainerState>
    /** Bilibili App only; returned Promise is NOT confirmation of mode change. */
    setContainerMode(req: SetContainerModeReq): Promise<void>
  }
}

interface Window {
  /** Optional because regular browser / dev mode has no Toy SDK. */
  toy?: ToySDK.Toy
}
/** Runtime-global SDK binding; check existence before access outside Toy. */
declare const toy: ToySDK.Toy | undefined
