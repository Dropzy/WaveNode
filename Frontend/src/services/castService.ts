import type { Music } from './api'

interface CastMediaMetadata {
	title?: string
	artist?: string
	albumName?: string
	images?: unknown[]
}

interface CastMediaInfo {
	metadata?: CastMediaMetadata
}

interface CastLoadRequest {
	currentTime: number
	autoplay: boolean
}

interface CastSession {
	loadMedia(request: CastLoadRequest): Promise<void>
	getCastDevice(): { friendlyName?: string }
}

interface CastContext {
	setOptions(options: { receiverApplicationId: string; autoJoinPolicy: string }): void
	requestSession(): Promise<void>
	getCurrentSession(): CastSession | null
}

interface CastFrameworkNamespace {
	CastContext: { getInstance(): CastContext }
}

interface ChromeCastNamespace {
	AutoJoinPolicy: { ORIGIN_SCOPED: string }
	Image: new (url: string) => unknown
	media: {
		DEFAULT_MEDIA_RECEIVER_APP_ID: string
		MediaInfo: new (url: string, contentType: string) => CastMediaInfo
		MusicTrackMediaMetadata: new () => CastMediaMetadata
		LoadRequest: new (mediaInfo: CastMediaInfo) => CastLoadRequest
	}
}

declare global {
	interface Window {
		__onGCastApiAvailable?: (available: boolean) => void
		cast?: { framework: CastFrameworkNamespace }
		chrome?: { cast?: ChromeCastNamespace }
	}
}

class CastService {
	private readyPromise: Promise<boolean> | null = null

	isSupported(): boolean {
		return 'chrome' in window && !/iPad|iPhone|iPod/.test(navigator.userAgent)
	}

	private initialize(): Promise<boolean> {
		if (this.readyPromise) return this.readyPromise
		this.readyPromise = new Promise(resolve => {
			if (!this.isSupported()) { resolve(false); return }
			window.__onGCastApiAvailable = available => {
				if (!available || !window.cast || !window.chrome?.cast) { resolve(false); return }
				window.cast.framework.CastContext.getInstance().setOptions({
					receiverApplicationId: window.chrome.cast.media.DEFAULT_MEDIA_RECEIVER_APP_ID,
					autoJoinPolicy: window.chrome.cast.AutoJoinPolicy.ORIGIN_SCOPED,
				})
				resolve(true)
			}
			const script = document.createElement('script')
			script.src = 'https://www.gstatic.com/cv/js/sender/v1/cast_sender.js?loadCastFramework=1'
			script.async = true
			script.onerror = () => resolve(false)
			document.head.appendChild(script)
		})
		return this.readyPromise
	}

	async cast(track: Music, mediaUrl: string, startTime: number): Promise<string> {
		if (!await this.initialize()) throw new Error('Google Cast is not available')
		if (!window.cast || !window.chrome?.cast) throw new Error('Google Cast is not available')
		const context = window.cast.framework.CastContext.getInstance()
		await context.requestSession()
		const session = context.getCurrentSession()
		if (!session) throw new Error('No Cast device was selected')
		const mediaInfo = new window.chrome.cast.media.MediaInfo(mediaUrl, 'audio/mpeg')
		const metadata = new window.chrome.cast.media.MusicTrackMediaMetadata()
		metadata.title = track.title
		metadata.artist = track.artist
		metadata.albumName = track.album
		if (track.image_url) metadata.images = [new window.chrome.cast.Image(track.image_url)]
		mediaInfo.metadata = metadata
		const request = new window.chrome.cast.media.LoadRequest(mediaInfo)
		request.currentTime = Math.max(0, startTime)
		request.autoplay = true
		await session.loadMedia(request)
		return session.getCastDevice().friendlyName || 'Google Cast device'
	}
}

export const castService = new CastService()
