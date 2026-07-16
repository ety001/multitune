const API_BASE = '/api'

class APIError extends Error {
  constructor(message, code) {
    super(message)
    this.name = 'APIError'
    this.code = code
  }
}

function toQuery(params) {
  const q = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') continue
    q.set(key, String(value))
  }
  const s = q.toString()
  return s ? '?' + s : ''
}

async function request(method, path, body = null) {
  const options = {
    method,
    headers: {},
  }
  if (body !== undefined && body !== null) {
    options.headers['Content-Type'] = 'application/json'
    options.body = JSON.stringify(body)
  }

  const res = await fetch(API_BASE + path, options)
  const data = await res.json()
  if (data.code !== 0) {
    throw new APIError(data.message || '请求失败', data.code)
  }
  return data.data
}

export const api = {
  get: (path, params = {}) => request('GET', path + toQuery(params)),
  post: (path, body) => request('POST', path, body),
  put: (path, body) => request('PUT', path, body),
  delete: (path) => request('DELETE', path),
}

export const identityApi = {
  list: () => api.get('/identities'),
  create: (body) => api.post('/identities', body),
  get: (id) => api.get('/identities/' + id),
  update: (id, body) => api.put('/identities/' + id, body),
  delete: (id) => api.delete('/identities/' + id),
  setDefault: (id) => api.post('/identities/' + id + '/default'),
}

export const playlistApi = {
  listByIdentity: (identityId) => api.get('/identities/' + identityId + '/playlists'),
  create: (identityId, body) => api.post('/identities/' + identityId + '/playlists', body),
  get: (id, params = {}) => api.get('/playlists/' + id, params),
  update: (id, body) => api.put('/playlists/' + id, body),
  delete: (id) => api.delete('/playlists/' + id),
  addSongs: (id, songIds) => api.post('/playlists/' + id + '/songs', { song_ids: songIds }),
  removeSong: (id, songId) => api.delete('/playlists/' + id + '/songs/' + songId),
  updateOrder: (id, songIds) => api.put('/playlists/' + id + '/songs/order', { song_ids: songIds }),
}

export const songApi = {
  list: (params = {}) => api.get('/songs', params),
  get: (id) => api.get('/songs/' + id),
}

export const fsApi = {
  sources: () => api.get('/fs/sources'),
  list: (path) => api.get('/fs/list', { path }),
  search: (params = {}) => api.get('/fs/search', params),
  scan: (path) => api.post('/scan', { path }),
}

export const scanApi = {
  createJob: (body) => api.post('/scan/jobs', body),
  getJob: (id) => api.get('/scan/jobs/' + id),
}

export const playbackApi = {
  get: (identityId) => api.get('/playback/' + identityId),
  save: (identityId, body) => api.post('/playback/' + identityId, body),
}

export { APIError }
