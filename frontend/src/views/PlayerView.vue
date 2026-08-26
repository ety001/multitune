<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { identityApi } from '../api/client'
import { usePlaylistStore } from '../stores/playlist'
import { usePlayerStore } from '../stores/player'

const props = defineProps({
  id: String,
})

const playlistStore = usePlaylistStore()
const playerStore = usePlayerStore()

const identity = ref(null)
const error = ref(null)
const resumeError = ref(null)
const progressRef = ref(null)

onMounted(async () => {
  await loadPlaylist()
  await nextTick()
  observeContainer()
})

watch(() => props.id, async () => {
  await loadPlaylist()
})

async function loadPlaylist() {
  try {
    error.value = null
    resumeError.value = null
    await playlistStore.fetchPlaylistDetail(props.id)
    if (playlistStore.currentPlaylist) {
      playerStore.setCurrentPlaylist(playlistStore.currentPlaylist)
      const identityData = await identityApi.get(playlistStore.currentPlaylist.identity_id)
      identity.value = identityData
      playerStore.setCurrentIdentity(identityData)
      await resumePlayback()
      // 恢复完成后首屏详情已由 resumePlaylist 的邻近预加载覆盖
      playerStore.ensureSongs(visibleIds.value).catch(() => {})
    }
  } catch (e) {
    error.value = e.message
  }
}

// 从歌单记忆点恢复播放；失败时展示错误和重试入口，不重载整页
async function resumePlayback() {
  if (!playlistStore.currentPlaylist) return
  resumeError.value = null
  try {
    await playerStore.resumePlaylist(playlistStore.currentPlaylist, identity.value)
  } catch (e) {
    resumeError.value = e.message || '记忆点加载失败'
  }
}

// ---- 虚拟列表 ----
// 行高固定（px），只渲染可视区 ±buffer 行；DOM 数量恒定，万级歌曲不卡
const ROW_HEIGHT = 44
const BUFFER = 8
const scrollContainer = ref(null)
const scrollTop = ref(0)
const viewportHeight = ref(600)

const songIds = computed(() => {
  const pl = playlistStore.currentPlaylist
  if (!pl) return []
  if (Array.isArray(pl.song_ids) && pl.song_ids.length > 0) return pl.song_ids
  if (Array.isArray(pl.songs)) return pl.songs.map((s) => s.id)
  return []
})

const total = computed(() => songIds.value.length)
const startIndex = computed(() => Math.max(0, Math.floor(scrollTop.value / ROW_HEIGHT) - BUFFER))
const endIndex = computed(() =>
  Math.min(total.value, Math.ceil((scrollTop.value + viewportHeight.value) / ROW_HEIGHT) + BUFFER)
)
const visibleIds = computed(() => songIds.value.slice(startIndex.value, endIndex.value))
const offsetY = computed(() => startIndex.value * ROW_HEIGHT)
const totalHeight = computed(() => total.value * ROW_HEIGHT)

function songOf(id) {
  return playerStore.songCache.get(id)
}

function isActiveId(id) {
  return !!playerStore.currentSong && playerStore.currentSong.id === id
}

// 滚动时节流触发详情按需加载（ensureSongs 内部有去重，未缺失时零请求）
let ensureTimer = null
function onScroll() {
  if (scrollContainer.value) {
    scrollTop.value = scrollContainer.value.scrollTop
  }
  if (ensureTimer) return
  ensureTimer = setTimeout(() => {
    ensureTimer = null
    const from = Math.max(0, startIndex.value - BUFFER)
    const to = Math.min(total.value, endIndex.value + BUFFER)
    playerStore.ensureSongs(songIds.value.slice(from, to)).catch(() => {})
  }, 150)
}

// 切歌后滚动定位到当前曲（居中）
watch(
  () => (playerStore.currentSong ? playerStore.currentSong.id : null),
  (id) => {
    if (!id || !scrollContainer.value) return
    const idx = songIds.value.indexOf(id)
    if (idx < 0) return
    const target = idx * ROW_HEIGHT + ROW_HEIGHT / 2 - viewportHeight.value / 2
    scrollContainer.value.scrollTo({ top: Math.max(0, target) })
  }
)

// 容器尺寸变化时刷新可视范围
let resizeObserver = null
function observeContainer() {
  if (!scrollContainer.value || typeof ResizeObserver === 'undefined') return
  resizeObserver = new ResizeObserver((entries) => {
    for (const entry of entries) {
      viewportHeight.value = entry.contentRect.height
    }
  })
  resizeObserver.observe(scrollContainer.value)
}

onUnmounted(() => {
  if (resizeObserver) resizeObserver.disconnect()
  if (ensureTimer) clearTimeout(ensureTimer)
})

// ---- 播放控制 ----
function playById(id) {
  if (playerStore.resuming) return
  const song = playerStore.songCache.get(id)
  if (!song) return
  playerStore.playSong(song, playlistStore.currentPlaylist, identity.value)
}

function confirmRemove(id, title) {
  if (!confirm('确定要从歌单中移除「' + (title || '未知歌曲') + '」吗？')) {
    return
  }
  playlistStore.removeSong(playlistStore.currentPlaylist.id, id)
}

function toggleMode() {
  const modes = ['order', 'random', 'single-loop']
  const idx = modes.indexOf(playerStore.mode)
  playerStore.setMode(modes[(idx + 1) % modes.length])
}

function onProgressClick(e) {
  if (!progressRef.value || !playerStore.duration) return
  const rect = progressRef.value.getBoundingClientRect()
  const ratio = (e.clientX - rect.left) / rect.width
  playerStore.seek(ratio * playerStore.duration)
}

function modeLabel(mode) {
  if (mode === 'order') return '顺序播放'
  if (mode === 'random') return '随机播放'
  if (mode === 'single-loop') return '单曲循环'
  return mode
}

function modeIcon(mode) {
  if (mode === 'order') return 'fa-arrow-right'
  if (mode === 'random') return 'fa-shuffle'
  if (mode === 'single-loop') return 'fa-rotate-right'
  return 'fa-question'
}
</script>

<template>
  <div class="player-page">
    <div class="page-header">
      <button class="btn btn-secondary" @click="$router.back()">← 返回</button>
      <div class="page-title">
        <h2>{{ playlistStore.currentPlaylist ? playlistStore.currentPlaylist.name : '播放器' }}</h2>
        <p class="hint">{{ identity ? identity.name : '' }} · 共 {{ total }} 首</p>
      </div>
    </div>

    <div v-if="error" class="error">{{ error }}</div>
    <div v-else-if="playlistStore.loading" class="empty">加载中...</div>

    <template v-else-if="playlistStore.currentPlaylist">
      <div v-if="playerStore.resuming" class="resume-hint">正在恢复上次播放…</div>
      <div v-if="resumeError" class="error resume-error">
        <span>{{ resumeError }}</span>
        <button class="btn btn-secondary btn-small" @click="resumePlayback">重试</button>
      </div>
      <div class="player-layout" :class="{ disabled: playerStore.resuming }">
        <!-- 窄屏时控制面板排在列表上方（CSS order），列表为固定高度滚动窗口 -->
        <div class="playlist-panel card">
          <div v-if="total === 0" class="empty">歌单为空，先去添加歌曲吧。</div>
          <div
            v-else
            ref="scrollContainer"
            class="song-scroll"
            @scroll.passive="onScroll"
          >
            <div class="song-spacer" :style="{ height: totalHeight + 'px' }">
              <div class="song-window" :style="{ transform: 'translateY(' + offsetY + 'px)' }">
                <div
                  v-for="(id, index) in visibleIds"
                  :key="id"
                  class="song-row"
                  :class="{ active: isActiveId(id) }"
                  @click="playById(id)"
                >
                  <span class="song-index">{{ startIndex + index + 1 }}</span>
                  <span class="song-title">{{ songOf(id) ? songOf(id).title : '加载中…' }}</span>
                  <button
                    class="btn btn-danger btn-small song-remove"
                    title="移除"
                    aria-label="移除"
                    @click.stop="confirmRemove(id, songOf(id) ? songOf(id).title : '')"
                  >
                    <i class="fas fa-trash-can"></i>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="control-panel card">
          <div class="now-playing">
            <div class="now-title">{{ playerStore.currentSong ? playerStore.currentSong.title : '未在播放' }}</div>
            <div class="now-artist">{{ playerStore.currentSong ? playerStore.currentSong.artist || '' : '点击歌曲开始播放' }}</div>
          </div>

          <div ref="progressRef" class="progress-bar" @click="onProgressClick">
            <div class="progress-fill" :style="{ width: (playerStore.duration ? (playerStore.currentTime / playerStore.duration) * 100 : 0) + '%' }"></div>
          </div>
          <div class="time-row">
            <span>{{ playerStore.currentTimeFormatted }}</span>
            <span>{{ playerStore.durationFormatted }}</span>
          </div>

          <div class="control-buttons">
            <button class="btn btn-secondary control-btn" @click="playerStore.prev" aria-label="上一曲">
              <i class="fas fa-backward-step"></i>
            </button>
            <button class="btn btn-primary control-btn play-btn" @click="playerStore.togglePlay" aria-label="播放/暂停">
              <i class="fas" :class="playerStore.isPlaying ? 'fa-pause' : 'fa-play'"></i>
            </button>
            <button class="btn btn-secondary control-btn" @click="playerStore.next" aria-label="下一曲">
              <i class="fas fa-forward-step"></i>
            </button>
            <button class="btn btn-secondary control-btn" @click="toggleMode" :title="modeLabel(playerStore.mode)">
              <i class="fas" :class="modeIcon(playerStore.mode)"></i>
            </button>
          </div>

          <div class="volume-row">
            <i class="fas fa-volume-high"></i>
            <input type="range" min="0" max="1" step="0.05" v-model.number="playerStore.volume" @input="playerStore.setVolume(playerStore.volume)" />
            <span>{{ Math.round(playerStore.volume * 100) }}%</span>
          </div>

          <div v-if="playerStore.error" class="error">{{ playerStore.error }}</div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}
.page-title h2 {
  font-size: 22px;
  margin-bottom: 4px;
}
.hint {
  color: #94a3b8;
  font-size: 14px;
}
.player-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 380px;
  gap: 24px;
  align-items: stretch;
}
.player-layout.disabled {
  opacity: 0.6;
  pointer-events: none;
}
.resume-hint {
  margin-bottom: 12px;
  color: #94a3b8;
  font-size: 14px;
}
.resume-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}
.playlist-panel {
  display: flex;
  flex-direction: column;
  min-height: 0;
}
/* 列表为视口内固定高度滚动窗口，页面整体不再随歌单长度增长 */
.song-scroll {
  height: calc(100vh - 220px);
  min-height: 320px;
  overflow-y: auto;
  overscroll-behavior: contain;
}
.song-spacer {
  position: relative;
}
.song-window {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  will-change: transform;
}
.song-row {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 44px;
  padding: 0 12px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.1);
  cursor: pointer;
  box-sizing: border-box;
}
.song-row:hover {
  background: rgba(148, 163, 184, 0.1);
}
.song-row.active {
  background: rgba(99, 102, 241, 0.15);
}
.song-index {
  width: 44px;
  flex: none;
  color: #94a3b8;
  font-size: 13px;
  text-align: right;
}
.song-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.song-remove {
  flex: none;
  visibility: hidden;
}
.song-row:hover .song-remove,
.song-row.active .song-remove {
  visibility: visible;
}
.btn-small {
  padding: 4px 10px;
  font-size: 12px;
}
.control-panel {
  position: sticky;
  top: 20px;
  align-self: flex-start;
}
.now-playing {
  margin-bottom: 20px;
  min-height: 70px;
}
.now-title {
  font-size: 18px;
  font-weight: 500;
  margin-bottom: 6px;
  word-break: break-all;
}
.now-artist {
  font-size: 14px;
  color: #94a3b8;
}
.progress-bar {
  height: 8px;
  background: rgba(148, 163, 184, 0.2);
  border-radius: 4px;
  cursor: pointer;
  margin-bottom: 8px;
}
.progress-fill {
  height: 100%;
  background: #6366f1;
  border-radius: 4px;
}
.time-row {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #94a3b8;
  margin-bottom: 20px;
}
.control-buttons {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
}
.control-btn {
  padding: 10px 16px;
  font-size: 16px;
}
.play-btn {
  padding: 14px 24px;
  font-size: 20px;
}
.volume-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.volume-row input[type='range'] {
  flex: 1;
}
@media (max-width: 900px) {
  .player-layout {
    grid-template-columns: 1fr;
  }
  /* 窄屏：控制面板置顶固定，列表窗口占剩余空间，无需悬浮 */
  .control-panel {
    position: static;
    order: -1;
    align-self: auto;
  }
  .song-scroll {
    height: 55vh;
    min-height: 240px;
  }
}
</style>
