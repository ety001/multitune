import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { playbackApi, playlistApi } from '../api/client'

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

  async function loadPlaybackState() {
    if (!currentIdentity.value) return
    try {
      const state = await playbackApi.get(currentIdentity.value.id)
      if (state && state.mode) {
        mode.value = state.mode
      }
      if (state && state.playlist_id) {
        const playlist = await playlistApi.get(state.playlist_id, { limit: 200 })
        currentPlaylist.value = playlist
        if (state.song_id) {
          const song = playlist.songs.find((s) => s.id === state.song_id)
          if (song) {
            currentSong.value = song
            audio.src = '/api/songs/' + song.id + '/stream'
            audio.currentTime = state.position || 0
            currentTime.value = audio.currentTime
          }
        }
      }
    } catch (e) {
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
      error.value = e.message
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

  async function savePlaybackState() {
    if (!currentIdentity.value) return
    try {
      await playbackApi.save(currentIdentity.value.id, {
        playlist_id: currentPlaylist.value ? currentPlaylist.value.id : '',
        song_id: currentSong.value ? currentSong.value.id : '',
        position: Math.floor(currentTime.value),
        mode: mode.value,
      })
    } catch (e) {
      // 播放状态保存失败不阻断播放
      console.error('保存播放状态失败', e)
    }
  }

  let saveTimer = null
  function startAutoSave() {
    stopAutoSave()
    saveTimer = setInterval(() => {
      if (isPlaying.value) savePlaybackState()
    }, 5000)
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
    error,
    currentTimeFormatted,
    durationFormatted,
    setVolume,
    setMode,
    setCurrentIdentity,
    setCurrentPlaylist,
    loadPlaybackState,
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
