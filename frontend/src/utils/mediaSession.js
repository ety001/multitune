// 媒体会话接入：让系统认到媒体会话，接收媒体硬件按键（方向盘/耳机线控/锁屏控制）。
// 两条通道，优先懒猫 WebShell 桥：
//   lzc-bridge — 懒猫客户端注入的全局 lzc_media_session（老车机 WebView 无
//                标准 API 时仍能建立原生媒体会话），回调经 window 的
//                lzc_media_session_event 事件派发，detail = { eventType, data }
//   standard   — 浏览器原生 navigator.mediaSession
//   none       — 均不可用，静默降级为无媒体按键
function msMode() {
  if (typeof lzc_media_session !== 'undefined') return 'lzc-bridge'
  if (typeof navigator !== 'undefined' && 'mediaSession' in navigator) return 'standard'
  return 'none'
}

let handlers = {}

export function initMediaSession(callbacks) {
  const mode = msMode()
  if (mode === 'none') return mode

  handlers = {
    play: callbacks.play,
    pause: callbacks.pause,
    nexttrack: callbacks.next,
    previoustrack: callbacks.prev,
    seekforward: () => callbacks.seekBy(10),
    seekbackward: () => callbacks.seekBy(-10),
    seekto: (data) => {
      if (data && typeof data.seekTime === 'number') callbacks.seekTo(data.seekTime)
    },
    stop: callbacks.pause,
  }

  window.addEventListener('lzc_media_session_event', (e) => {
    const detail = e.detail || {}
    const fn = handlers[detail.eventType]
    if (fn) fn(detail.data)
  })

  for (const [name, fn] of Object.entries(handlers)) {
    try {
      if (mode === 'lzc-bridge') {
        lzc_media_session.setActionHandler(name)
      } else {
        navigator.mediaSession.setActionHandler(name, fn)
      }
    } catch {
      // 该 action 不被支持，跳过
    }
  }
  return mode
}

export function msSetPlaybackState(state) {
  try {
    if (msMode() === 'lzc-bridge') {
      lzc_media_session.setPlaybackState(state)
    } else if ('mediaSession' in navigator) {
      navigator.mediaSession.playbackState = state
    }
  } catch {
    // 状态同步失败不影响播放
  }
}

export function msSetPositionState(duration, position, rate) {
  try {
    if (msMode() === 'lzc-bridge') {
      lzc_media_session.setPositionState(JSON.stringify({ duration, position, playbackRate: rate }))
    } else if ('mediaSession' in navigator && typeof navigator.mediaSession.setPositionState === 'function') {
      navigator.mediaSession.setPositionState({ duration, position, playbackRate: rate })
    }
  } catch {
    // 进度同步失败不影响播放
  }
}

// 换歌时更新元信息（标题/歌手/封面），供锁屏与系统媒体控制显示
export function msSetMetadata(song) {
  const mode = msMode()
  if (mode === 'none') return
  const meta = {
    title: (song && song.title) || '未知歌曲',
    artist: (song && song.artist) || '',
    album: 'MultiTune',
  }
  if (song && song.id) {
    // 原生侧需要完整 URL 才能拉封面
    meta.artwork = [
      { src: location.origin + '/api/songs/' + encodeURIComponent(song.id) + '/cover?size=thumb', sizes: '256x256', type: 'image/webp' },
    ]
  }
  try {
    if (mode === 'lzc-bridge') {
      lzc_media_session.setMetadata(JSON.stringify(meta))
    } else {
      navigator.mediaSession.metadata = new MediaMetadata(meta)
    }
  } catch {
    // 元信息设置失败不影响播放
  }
}
