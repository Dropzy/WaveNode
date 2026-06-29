const databaseName = 'wavenode-podcast-downloads'
const storeName = 'episodes'

type DownloadedEpisode = {
	id: string
	blob: Blob
	downloadedAt: number
}

const openDatabase = (): Promise<IDBDatabase> => new Promise((resolve, reject) => {
	const request = indexedDB.open(databaseName, 1)
	request.onupgradeneeded = () => {
		if (!request.result.objectStoreNames.contains(storeName)) {
			request.result.createObjectStore(storeName, { keyPath: 'id' })
		}
	}
	request.onsuccess = () => resolve(request.result)
	request.onerror = () => reject(request.error)
})

const transact = async <T>(mode: IDBTransactionMode, operation: (store: IDBObjectStore) => IDBRequest<T>): Promise<T> => {
	const database = await openDatabase()
	try {
		return await new Promise<T>((resolve, reject) => {
			const transaction = database.transaction(storeName, mode)
			const request = operation(transaction.objectStore(storeName))
			request.onsuccess = () => resolve(request.result)
			request.onerror = () => reject(request.error)
		})
	} finally {
		database.close()
	}
}

export const downloadPodcastEpisode = async (id: string, audioUrl: string): Promise<void> => {
	const response = await fetch(audioUrl)
	if (!response.ok) throw new Error(`Episode download failed (${response.status})`)
	const blob = await response.blob()
	if (!blob.size) throw new Error('Episode download was empty')
	await transact('readwrite', store => store.put({ id, blob, downloadedAt: Date.now() } satisfies DownloadedEpisode))
}

export const hasPodcastDownload = async (id: string): Promise<boolean> => Boolean(await transact('readonly', store => store.getKey(id)))

export const getPodcastDownloadUrl = async (id: string): Promise<string | null> => {
	const episode = await transact<DownloadedEpisode | undefined>('readonly', store => store.get(id))
	return episode?.blob ? URL.createObjectURL(episode.blob) : null
}

export const removePodcastDownload = async (id: string): Promise<void> => {
	await transact('readwrite', store => store.delete(id))
}
