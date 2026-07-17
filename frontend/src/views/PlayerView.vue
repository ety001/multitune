<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { identityApi } from '../api/client'
import { usePlaylistStore } from '../stores/playlist'
import { usePlayerStore } from '../stores/player'

const props = defineProps({
  id: String,
})

const router = useRouter()
const playlistStore = usePlaylistStore()
const playerStore = usePlayerStore()

const identity = ref(null)
const error = ref(null)
const resumeError = ref(null)
const progressRef = ref(null)

onMounted(async () => {
  await loadPlaylist()
})

watch(() => props.id, async () => {
  await loadPlaylist()
})

async function loadPlaylist() {
  try {
    error.value = null
    resumeError.value = null
    await playlistStore.fetchPlaylistDetail(props.id, 200, 0)
    if (playlistStore.currentPlaylist) {
      playerStore.setCurrentPlaylist(playlistStore.currentPlaylist)
      const identityData = await identityApi.get(playlistStore.currentPlaylist.identity_id)
      identity.value = identityData
      playerStore.setCurrentIdentity(identityData)
      await resumePlayback()
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

function isActiveSong(song) {
  return playerStore.currentSong && playerStore.currentSong.id === song.id
}

function playSong(song) {
  if (playerStore.resuming) return
  playerStore.playSong(song, playlistStore.currentPlaylist, identity.value)
}

function confirmRemove(song) {
  if (!confirm('确定要从歌单中移除「' + (song.title || '未知歌曲') + '」吗？')) {
    return
  }
  playlistStore.removeSong(playlistStore.currentPlaylist.id, song.id)
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
        <p class="hint">{{ identity ? identity.name : '' }} · 共 {{ playlistStore.currentPlaylist ? playlistStore.currentPlaylist.song_count : 0 }} 首</p>
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
      <div class="playlist-panel card">
        <div v-if="playlistStore.currentPlaylist.songs.length === 0" class="empty">歌单为空，先去添加歌曲吧。</div>
        <table v-else class="song-table">
          <thead>
            <tr>
              <th>#</th>
              <th>歌曲</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(song, index) in playlistStore.currentPlaylist.songs"
              :key="song.id"
              :class="{ active: isActiveSong(song) }"
              @click="playSong(song)"
            >
              <td>{{ index + 1 }}</td>
              <td>{{ song.title }}</td>
              <td>
                <button class="btn btn-danger btn-small" @click.stop="confirmRemove(song)">移除</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="control-panel card">
        <div class="now-playing">
          <div class="now-title">{{ playerStore.currentSong ? playerStore.currentSong.title : '未在播放' }}</div>
          <div class="now-artist">{{ playerStore.currentSong ? playerStore.currentSong.artist || '' : '点击左侧歌曲开始播放' }}</div>
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
  align-items: flex-start;
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
  min-height: 300px;
}
.song-table {
  width: 100%;
  border-collapse: collapse;
}
.song-table th,
.song-table td {
  text-align: left;
  padding: 10px 12px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.1);
}
.song-table th {
  color: #94a3b8;
  font-size: 13px;
  font-weight: 500;
}
.song-table tbody tr {
  cursor: pointer;
}
.song-table tbody tr:hover {
  background: rgba(148, 163, 184, 0.1);
}
.song-table tbody tr.active {
  background: rgba(99, 102, 241, 0.15);
}
.btn-small {
  padding: 4px 10px;
  font-size: 12px;
}
.control-panel {
  position: sticky;
  top: 20px;
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
.volume-row input[type="range"] {
  flex: 1;
}
@media (max-width: 900px) {
  .player-layout {
    grid-template-columns: 1fr;
  }
  .control-panel {
    position: static;
  }
}
</style>
