import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { playbackApi } from '../api/client'

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
  }

  // 进入歌单时恢复播放：歌单记忆点决定起始曲目和位置，身份记忆点决定播放模式
  let resumeSeq = 0
  async function resumePlaylist(playlist, identity) {
    if (!playlist) return
    if (identity) currentIdentity.value = identity
    currentPlaylist.value = playlist
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

      const songs = playlist.songs || []
      if (songs.length === 0) return

      let startSong = songs[0]
      let startPosition = 0
      if (progress && progress.song_id) {
        const found = songs.find((s) => s.id === progress.song_id)
        if (found) {
          startSong = found
          startPosition = progress.position || 0
        }
      }

      currentSong.value = startSong
      audio.src = '/api/songs/' + startSong.id + '/stream'
      audio.currentTime = startPosition
      currentTime.value = startPosition
      await tryPlay()
    } finally {
      if (seq === resumeSeq) {
        resuming.value = false
      }
    }
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
    if (playlist) currentPlaylist.value = playlist
    if (identity) currentIdentity.value = identity
    currentSong.value = song
    error.value = null

    try {
      loading.value = true
      audio.src = '/api/songs/' + song.id + '/stream'
      audio.currentTime = startPosition || 0
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
    if (!currentPlaylist.value || !currentSong.value) return -1
    return currentPlaylist.value.songs.findIndex((s) => s.id === currentSong.value.id)
  }

  function next() {
    if (!currentPlaylist.value || currentPlaylist.value.songs.length === 0) return
    const len = currentPlaylist.value.songs.length
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

    const song = currentPlaylist.value.songs[idx]
    if (song) playSong(song)
  }

  function prev() {
    if (!currentPlaylist.value || currentPlaylist.value.songs.length === 0) return
    const len = currentPlaylist.value.songs.length
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

    const song = currentPlaylist.value.songs[idx]
    if (song) playSong(song)
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
