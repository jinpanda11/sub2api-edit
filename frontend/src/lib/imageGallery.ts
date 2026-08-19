/**
 * 画图工作台画廊：IndexedDB 本地持久化（生成历史 + 收藏）。
 * 数据完全本地存储，不经过后端。
 */

export interface GalleryImage {
  dataUrl: string
  url?: string
  revised_prompt?: string
}

export interface GalleryRecord {
  id: string
  prompt: string
  model: string
  size?: string
  quality?: string
  mode: 'generation' | 'edit'
  images: GalleryImage[]
  favorite: boolean
  createdAt: number
}

const DB_NAME = 'sub2api-image-playground'
const DB_VERSION = 1
const STORE = 'records'

let dbPromise: Promise<IDBDatabase> | null = null

function openDB(): Promise<IDBDatabase> {
  if (dbPromise) return dbPromise
  dbPromise = new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION)
    request.onupgradeneeded = () => {
      const db = request.result
      if (!db.objectStoreNames.contains(STORE)) {
        db.createObjectStore(STORE, { keyPath: 'id' })
      }
    }
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
  })
  return dbPromise
}

async function withStore<T>(
  mode: IDBTransactionMode,
  fn: (store: IDBObjectStore) => IDBRequest<T>,
): Promise<T> {
  const db = await openDB()
  return new Promise<T>((resolve, reject) => {
    const tx = db.transaction(STORE, mode)
    const request = fn(tx.objectStore(STORE))
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
  })
}

export async function galleryAdd(record: GalleryRecord): Promise<void> {
  await withStore('readwrite', (store) => store.put(record))
}

export async function galleryList(): Promise<GalleryRecord[]> {
  const all = await withStore<GalleryRecord[]>('readonly', (store) => store.getAll())
  return all.sort((a, b) => b.createdAt - a.createdAt)
}

export async function galleryToggleFavorite(id: string, favorite: boolean): Promise<void> {
  const db = await openDB()
  await new Promise<void>((resolve, reject) => {
    const tx = db.transaction(STORE, 'readwrite')
    const store = tx.objectStore(STORE)
    const request = store.get(id)
    request.onsuccess = () => {
      const record = request.result as GalleryRecord | undefined
      if (record) {
        record.favorite = favorite
        store.put(record)
      }
    }
    tx.oncomplete = () => resolve()
    tx.onerror = () => reject(tx.error)
  })
}

export async function galleryRemove(id: string): Promise<void> {
  await withStore('readwrite', (store) => store.delete(id))
}
