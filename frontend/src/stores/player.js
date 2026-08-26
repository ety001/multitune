import { defineStore } from 'pinia'
import { ref, reactive, computed } from 'vue'
import { playbackApi, songsApi } from '../api/client'

const VOLUME_KEY = 'multitune_volume'

function formatTime(seconds) {
  if (!isFinite(seconds) || seconds < 0) return '0:00'
  const s = Math.floor(seconds)
  const m = Math.floor(s / 60)
  const h = Math.floor(m / 60)
  const rem = s % 60
  const pad = rem < 10 ? '0' + rem : String(rem)
  if (h > 0) return `${h}:${m < 10 ? '0' + m : m}:${pad}`
  return `${m}:${pad}`
}

export const usePlayerStore = defineStore('player', () => {
  const currentIdentity = ref(null)
  const currentPlaylist = ref(null)
  const currentSong = ref(null)
  const isPlaying = ref(false)
  const currentTime = ref(0)
  const duration = ref(0)
  const mode = ref('order')
  const volume = ref(1)
  const loading = ref(false)
  const resuming = ref(false)
  const error = ref(null)

  // 歌曲详情缓存：歌单详情只提供全量 song_ids，详情按需经 /songs/batch 加载
  const songCache = reactive(new Map())
  const pendingIds = new Set()

  const audio = new Audio()
  audio.preload = 'metadata'

  const currentTimeFormatted = computed(() => formatTime(currentTime.value))
  const durationFormatted = computed(() => formatTime(duration.value))

  function restoreVolume() {
    const saved = localStorage.getItem(VOLUME_KEY)
    const v = saved !== null ? parseFloat(saved) : 1
    if (!isNaN(v)) {
      volume.value = Math.max(0, Math.min(1, v))
      audio.volume = volume.value
    }
  }
  restoreVolume()

  function setVolume(v) {
    const value = Math.max(0, Math.min(1, v))
    volume.value = value
    audio.volume = value
    localStorage.setItem(VOLUME_KEY, String(value))
  }

  function setMode(m) {
    if (['order', 'random', 'single-loop'].includes(m)) {
      mode.value = m
      savePlaybackState()
    }
  }

  function setCurrentIdentity(identity) {
    currentIdentity.value = identity
  }

  function setCurrentPlaylist(playlist) {
    currentPlaylist.value = playlist
    // 歌单详情第一页 songs 直接预热缓存，首屏零额外请求
    if (playlist && Array.isArray(playlist.songs)) {
      for (const s of playlist.songs) {
        if (s) songCache.set(s.id, s)
      }
    }
  }

  // 当前歌单的全量歌曲 ID 有序列表
  const songIds = computed(() => {
    const pl = currentPlaylist.value
    if (!pl) return []
    if (Array.isArray(pl.song_ids) && pl.song_ids.length > 0) return pl.song_ids
    if (Array.isArray(pl.songs)) return pl.songs.map((s) => s.id)
    return []
  })

  // ensureSongs 批量加载缺失的歌曲详情（分块，每块最多 100，与后端限制一致）
  async function ensureSongs(ids) {
    const missing = []
    for (const id of ids) {
      if (!id || songCache.has(id) || pendingIds.has(id)) continue
      pendingIds.add(id)
      missing.push(id)
    }
    if (missing.length === 0) return
    try {
      for (let i = 0; i < missing.length; i += 100) {
        const chunk = missing.slice(i, i + 100)
        const data = await songsApi.batch(chunk)
        for (const s of data.songs || []) {
          songCache.set(s.id, s)
        }
      }
    } finally {
      for (const id of missing) pendingIds.delete(id)
    }
  }

  // 进入歌单时恢复播放：歌单记忆点决定起始曲目和位置，身份记忆点决定播放模式
  let resumeSeq = 0
  async function resumePlaylist(playlist, identity) {
    if (!playlist) return
    if (identity) currentIdentity.value = identity
    setCurrentPlaylist(playlist)
    resumeSeq += 1
    const seq = resumeSeq
    resuming.value = true
    error.value = null
    try {
      const results = await Promise.allSettled([
        playbackApi.getPlaylistProgress(playlist.id),
        currentIdentity.value ? playbackApi.get(currentIdentity.value.id) : Promise.resolve(null),
      ])
      if (seq !== resumeSeq) return // 已被新的恢复取代

      if (results[0].status === 'rejected') {
        // 歌单记忆点获取失败：抛出由 UI 层展示错误和重试入口
        throw results[0].reason
      }
      const progress = results[0].value
      const state = results[1].status === 'fulfilled' ? results[1].value : null

      if (state && state.mode && ['order', 'random', 'single-loop'].includes(state.mode)) {
        mode.value = state.mode
      }

      const ids = songIds.value
      if (ids.length === 0) return

      let startIdx = 0
      let startPosition = 0
      if (progress && progress.song_id) {
        const idx = ids.indexOf(progress.song_id)
        if (idx >= 0) {
          startIdx = idx
          startPosition = progress.position || 0
        }
      }

      // 当前曲与邻近几首预加载详情，起播与首次切歌都无需额外等待
      const nearby = []
      for (let i = Math.max(0, startIdx - 1); i <= Math.min(ids.length - 1, startIdx + 2); i++) {
        nearby.push(ids[i])
      }
      await ensureSongs(nearby)
      if (seq !== resumeSeq) return // 加载期间已被新的恢复取代

      const startSong = songCache.get(ids[startIdx])
      if (!startSong) return
      currentSong.value = startSong
      audio.src = '/api/songs/' + startSong.id + '/stream'
      // 续播：先跳转到记忆位置再播放。若立即 play()，浏览器会先从文件头
      // 缓冲、跳转时再放弃并重新发 Range 请求，弱网下双请求抢占带宽
      const src = audio.src
      await seekBeforePlay(startPosition, src)
      if (seq !== resumeSeq) return // 等待期间已被新的恢复取代
      currentTime.value = audio.currentTime || startPosition
      await tryPlay()
    } finally {
      if (seq === resumeSeq) {
        resuming.value = false
      }
    }
  }

  // 续播辅助：等元数据加载后设置 currentTime，再由调用方 play。
  // 元数据 8 秒未到达（弱网）时放弃等待，从头播放总好过一直等待。
  // expectedSrc 用于守卫：等待期间被切歌取代时不再设置过期的跳转位置。
  function seekBeforePlay(startPosition, expectedSrc) {
    if (!startPosition || startPosition <= 0) return Promise.resolve()
    return new Promise((resolve) => {
      let settled = false
      const done = () => {
        if (settled) return
        settled = true
        audio.removeEventListener('loadedmetadata', onMeta)
        resolve()
      }
      const onMeta = () => {
        if (audio.src !== expectedSrc) {
          done()
          return
        }
        try {
          audio.currentTime = startPosition
        } catch {
          // 部分格式不支持精确 seek，从头播放
        }
        done()
      }
      audio.addEventListener('loadedmetadata', onMeta)
      setTimeout(done, 8000)
    })
  }

  async function tryPlay() {
    try {
      await audio.play()
      isPlaying.value = true
      startAutoSave()
    } catch (e) {
      isPlaying.value = false
      // 浏览器自动播放限制：保持就绪暂停态，不报错
      if (e && e.name === 'NotAllowedError') return
      error.value = e.message
    }
  }

  async function playSong(song, playlist = null, identity = null, startPosition = 0) {
    if (!song) return
    if (playlist) setCurrentPlaylist(playlist)
    if (identity) currentIdentity.value = identity
    currentSong.value = song
    songCache.set(song.id, song)
    error.value = null

    try {
      loading.value = true
      audio.src = '/api/songs/' + song.id + '/stream'
      const src = audio.src
      await seekBeforePlay(startPosition || 0, src)
      currentTime.value = audio.currentTime || startPosition || 0
      await audio.play()
      isPlaying.value = true
      startAutoSave()
    } catch (e) {
      // 浏览器自动播放限制时保持暂停就绪，不作为错误展示
      if (!(e && e.name === 'NotAllowedError')) {
        error.value = e.message
      }
      isPlaying.value = false
    } finally {
      loading.value = false
    }
  }

  function togglePlay() {
    if (!currentSong.value) return
    if (isPlaying.value) {
      audio.pause()
    } else {
      audio.play().catch((e) => {
        error.value = e.message
      })
    }
  }

  function pause() {
    if (audio.paused) return
    audio.pause()
  }

  function seek(t) {
    if (!isFinite(t)) return
    audio.currentTime = Math.max(0, Math.min(t, duration.value || t))
  }

  function getCurrentIndex() {
    if (!currentSong.value) return -1
    return songIds.value.indexOf(currentSong.value.id)
  }

  // playByIndex 索引对应的歌曲（详情缺失时按需加载后播放）
  async function playByIndex(idx) {
    const id = songIds.value[idx]
    if (!id) return
    await ensureSongs([id])
    const song = songCache.get(id)
    if (song) playSong(song)
  }

  function next() {
    const len = songIds.value.length
    if (len === 0) return
    let idx = getCurrentIndex()

    if (mode.value === 'random') {
      let nextIdx = idx
      if (len > 1) {
        while (nextIdx === idx) {
          nextIdx = Math.floor(Math.random() * len)
        }
      }
      idx = nextIdx
    } else {
      idx = idx + 1
      if (idx >= len) {
        idx = 0
      }
    }

    playByIndex(idx)
  }

  function prev() {
    const len = songIds.value.length
    if (len === 0) return
    let idx = getCurrentIndex()

    if (mode.value === 'random') {
      let nextIdx = idx
      if (len > 1) {
        while (nextIdx === idx) {
          nextIdx = Math.floor(Math.random() * len)
        }
      }
      idx = nextIdx
    } else {
      idx = idx - 1
      if (idx < 0) {
        idx = len - 1
      }
    }

    playByIndex(idx)
  }

  function onEnded() {
    if (mode.value === 'single-loop') {
      audio.currentTime = 0
      audio.play()
    } else {
      next()
    }
  }

  // includeMode 为 true 时附带播放模式（模式切换、暂停等关键节点）；
  // 周期上报只发 3 个字段以压缩体积
  async function savePlaybackState(includeMode = true) {
    if (!currentIdentity.value) return
    const body = {
      playlist_id: currentPlaylist.value ? currentPlaylist.value.id : '',
      song_id: currentSong.value ? currentSong.value.id : '',
      position: Math.floor(currentTime.value),
    }
    if (includeMode) {
      body.mode = mode.value
    }
    try {
      await playbackApi.save(currentIdentity.value.id, body)
    } catch (e) {
      // 播放状态保存失败不阻断播放
      console.error('保存播放状态失败', e)
    }
  }

  let saveTimer = null
  function startAutoSave() {
    stopAutoSave()
    saveTimer = setInterval(() => {
      if (isPlaying.value) savePlaybackState(false)
    }, 10000)
  }
  function stopAutoSave() {
    if (saveTimer) {
      clearInterval(saveTimer)
      saveTimer = null
    }
  }

  audio.addEventListener('loadedmetadata', () => {
    duration.value = audio.duration || 0
  })
  audio.addEventListener('timeupdate', () => {
    currentTime.value = audio.currentTime || 0
  })
  audio.addEventListener('play', () => {
    isPlaying.value = true
  })
  audio.addEventListener('pause', () => {
    isPlaying.value = false
    savePlaybackState()
  })
  audio.addEventListener('ended', () => {
    onEnded()
  })
  audio.addEventListener('error', () => {
    error.value = '音频加载失败或文件不可用'
    isPlaying.value = false
  })

  return {
    currentIdentity,
    currentPlaylist,
    currentSong,
    isPlaying,
    currentTime,
    duration,
    mode,
    volume,
    loading,
    resuming,
    error,
    currentTimeFormatted,
    durationFormatted,
    setVolume,
    setMode,
    setCurrentIdentity,
    setCurrentPlaylist,
    songIds,
    songCache,
    ensureSongs,
    resumePlaylist,
    playSong,
    togglePlay,
    pause,
    seek,
    next,
    prev,
    savePlaybackState,
    startAutoSave,
    stopAutoSave,
  }
})
