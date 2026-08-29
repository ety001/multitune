<script setup>
import { ref, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { identityApi, playbackApi, playlistApi } from '../api/client'
import { usePlaylistStore } from '../stores/playlist'
import { usePlayerStore } from '../stores/player'

const props = defineProps({
  id: String,
})

const router = useRouter()
const playlistStore = usePlaylistStore()
const playerStore = usePlayerStore()

const identity = ref(null)
const newName = ref('')
const showCreateModal = ref(false)
const editing = ref(null)
const deleting = ref(null)
const error = ref(null)
const lastPlaylistId = ref('')
const memoryError = ref(null)

// ===== 歌单搜索：总数不超过阈值时前端过滤，超过时后端搜索 =====
const searchQuery = ref('')
const searchResults = ref(null) // 后端搜索结果；null 表示未在搜索态
const backendMode = computed(
  () => playlistStore.totalCount > playlistStore.windowThreshold
)
const displayed = computed(() => {
  if (searchResults.value !== null) return searchResults.value
  if (backendMode.value) return playlistStore.playlists
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return playlistStore.playlists
  return playlistStore.playlists.filter((p) => (p.name || '').toLowerCase().includes(q))
})

let searchTimer = null
function onSearchInput() {
  if (!backendMode.value) return // 前端过滤由 computed 直接生效
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(async () => {
    const q = searchQuery.value.trim()
    if (!q) {
      searchResults.value = null
      return
    }
    try {
      const data = await playlistApi.listByIdentity(props.id, { q })
      searchResults.value = data.items || []
    } catch {
      // 搜索失败保留当前展示
    }
  }, 300)
}
onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
})

// 增删改后回到未搜索态并重拉全量列表
async function refreshList() {
  searchResults.value = null
  searchQuery.value = ''
  await playlistStore.fetchPlaylists(props.id)
}

// ===== 窗口化渲染：超过阈值时只渲染可视窗口（行对齐的网格切片） =====
const CARD_MIN_W = 260
const CARD_GAP = 16
const CARD_H = 104
const BUFFER_ROWS = 2

const windowed = computed(() => displayed.value.length > playlistStore.windowThreshold)
const scrollContainer = ref(null)
const scrollTop = ref(0)
const viewportHeight = ref(600)
const columns = ref(3)

const rowStride = CARD_H + CARD_GAP
const totalRows = computed(() =>
  windowed.value ? Math.ceil(displayed.value.length / Math.max(1, columns.value)) : 0
)
const spacerHeight = computed(() => (windowed.value ? totalRows.value * rowStride - CARD_GAP : 0))

const startRow = computed(() => {
  if (!windowed.value) return 0
  return Math.max(0, Math.floor(scrollTop.value / rowStride) - BUFFER_ROWS)
})
const endRow = computed(() => {
  if (!windowed.value) return 0
  const visible = Math.ceil(viewportHeight.value / rowStride) + BUFFER_ROWS * 2
  return Math.min(totalRows.value, startRow.value + visible)
})
const renderedItems = computed(() => {
  if (!windowed.value) return displayed.value
  return displayed.value.slice(startRow.value * columns.value, endRow.value * columns.value)
})
const offsetY = computed(() => (windowed.value ? startRow.value * rowStride : 0))

function onScroll() {
  if (scrollContainer.value) scrollTop.value = scrollContainer.value.scrollTop
}

let resizeObserver = null
function measure() {
  const el = scrollContainer.value
  if (!el) return
  viewportHeight.value = el.clientHeight || 600
  columns.value = Math.max(1, Math.floor((el.clientWidth + CARD_GAP) / (CARD_MIN_W + CARD_GAP)))
}
onMounted(async () => {
  await loadIdentity()
  await playlistStore.fetchPlaylists(props.id)
  fetchLastPlayed()
  await nextTick()
  measure()
  if (typeof ResizeObserver !== 'undefined' && scrollContainer.value) {
    resizeObserver = new ResizeObserver(measure)
    resizeObserver.observe(scrollContainer.value)
  }
})
onBeforeUnmount(() => {
  if (resizeObserver) resizeObserver.disconnect()
})

async function loadIdentity() {
  try {
    identity.value = await identityApi.get(props.id)
    playerStore.setCurrentIdentity(identity.value)
  } catch (e) {
    error.value = e.message
  }
}

// 拉取身份记忆点，用于在歌单卡片上标注"上次播放"；失败不阻塞列表
async function fetchLastPlayed() {
  memoryError.value = null
  try {
    const state = await playbackApi.get(props.id)
    lastPlaylistId.value = state && state.playlist_id ? state.playlist_id : ''
  } catch (e) {
    lastPlaylistId.value = ''
    memoryError.value = e.message || '记忆点加载失败'
  }
}

function openCreateModal() {
  newName.value = ''
  showCreateModal.value = true
}

function closeCreateModal() {
  showCreateModal.value = false
}

async function createPlaylist() {
  const name = newName.value.trim()
  if (!name) return
  await playlistStore.createPlaylist(props.id, name)
  closeCreateModal()
  refreshList()
}

function startEdit(playlist) {
  editing.value = {
    id: playlist.id,
    name: playlist.name,
  }
}

async function saveEdit() {
  if (!editing.value) return
  const name = editing.value.name.trim()
  if (!name) return
  await playlistStore.updatePlaylist(editing.value.id, { name })
  editing.value = null
  refreshList()
}

function startDelete(playlist) {
  deleting.value = { ...playlist }
}

function closeDeleteModal() {
  deleting.value = null
}

async function confirmDelete() {
  if (!deleting.value) return
  await playlistStore.deletePlaylist(deleting.value.id)
  closeDeleteModal()
  refreshList()
}

function goPlayer(playlist) {
  router.push('/playlists/' + playlist.id)
}
</script>

<template>
  <div>
    <div class="page-header">
      <button class="btn btn-secondary" @click="router.push('/')">&larr; 返回身份列表</button>
      <input
        v-model="searchQuery"
        class="playlist-search"
        type="text"
        placeholder="搜索歌单"
        @input="onSearchInput"
      />
      <button class="btn btn-primary" @click="openCreateModal">+ 新建歌单</button>
    </div>

    <div class="page-title">
      <h2>{{ identity ? identity.name : '歌单管理' }}</h2>
      <p class="hint">选择歌单进入播放器，或在此管理该身份下的歌单。最近播放的歌单排在前面。</p>
    </div>

    <div v-if="error" class="error">{{ error }}</div>

    <div v-if="memoryError" class="warn-bar">
      记忆点加载失败，无法标注上次播放
      <a class="retry-link" @click="fetchLastPlayed">重试</a>
    </div>

    <div v-if="playlistStore.loading" class="empty">加载中...</div>
    <div v-else-if="playlistStore.playlists.length === 0" class="empty">
      该身份下还没有歌单，去<a @click="router.push('/file-browser')">文件浏览器</a>添加歌曲吧。
    </div>
    <div v-else-if="displayed.length === 0" class="empty">没有匹配「{{ searchQuery }}」的歌单</div>

    <div
      v-else
      ref="scrollContainer"
      class="playlist-scroll"
      :class="{ windowed }"
      @scroll.passive="onScroll"
    >
      <div :style="windowed ? { height: spacerHeight + 'px', position: 'relative' } : {}">
        <div
          class="playlist-grid"
          :class="{ windowed }"
          :style="windowed ? { transform: 'translateY(' + offsetY + 'px)' } : {}"
        >
          <div v-for="playlist in renderedItems" :key="playlist.id" class="playlist-card card">
            <div class="playlist-info" @click="goPlayer(playlist)">
              <div class="playlist-name">
                {{ playlist.name }}
                <span v-if="playlist.id === lastPlaylistId" class="last-played-badge">上次播放</span>
              </div>
              <div class="playlist-count">{{ playlist.song_count || 0 }} 首歌曲</div>
            </div>
            <div class="playlist-actions" @click.stop>
              <button class="btn btn-secondary" @click="startEdit(playlist)">编辑</button>
              <button class="btn btn-danger" @click="startDelete(playlist)">删除</button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 新建歌单弹层 -->
    <div v-if="showCreateModal" class="modal">
      <div class="modal-content card">
        <div class="modal-header">
          <h3>新建歌单</h3>
          <button class="modal-close" @click="closeCreateModal">&times;</button>
        </div>
        <div class="modal-body">
          <div class="form-row">
            <label>歌单名称</label>
            <input v-model="newName" type="text" placeholder="例如：驾驶模式" @keyup.enter="createPlaylist" />
          </div>
        </div>
        <div class="modal-actions">
          <button class="btn btn-secondary" @click="closeCreateModal">取消</button>
          <button class="btn btn-primary" :disabled="!newName.trim()" @click="createPlaylist">创建</button>
        </div>
      </div>
    </div>

    <!-- 删除确认弹层 -->
    <div v-if="deleting" class="modal">
      <div class="modal-content card">
        <div class="modal-header">
          <h3>删除歌单</h3>
          <button class="modal-close" @click="closeDeleteModal">&times;</button>
        </div>
        <div class="modal-body">
          <p class="confirm-text">
            确定要删除歌单 <strong>{{ deleting.name }}</strong> 吗？
          </p>
          <p class="confirm-hint">删除后歌单中的所有歌曲关联将被移除，但歌曲文件不会被删除。</p>
        </div>
        <div class="modal-actions">
          <button class="btn btn-secondary" @click="closeDeleteModal">取消</button>
          <button class="btn btn-danger" @click="confirmDelete">确认删除</button>
        </div>
      </div>
    </div>

    <!-- 编辑歌单弹层 -->
    <div v-if="editing" class="modal">
      <div class="modal-content card">
        <div class="modal-header">
          <h3>编辑歌单</h3>
          <button class="modal-close" @click="editing = null">&times;</button>
        </div>
        <div class="modal-body">
          <div class="form-row">
            <label>歌单名称</label>
            <input v-model="editing.name" type="text" placeholder="歌单名称" @keyup.enter="saveEdit" />
          </div>
        </div>
        <div class="modal-actions">
          <button class="btn btn-secondary" @click="editing = null">取消</button>
          <button class="btn btn-primary" :disabled="!editing.name.trim()" @click="saveEdit">保存</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}
.playlist-search {
  flex: 1;
  min-width: 180px;
  max-width: 360px;
  padding: 8px 14px;
  background: rgba(148, 163, 184, 0.08);
  border: 1px solid rgba(148, 163, 184, 0.25);
  border-radius: 8px;
  color: #e2e8f0;
  font-size: 14px;
  outline: none;
}
.playlist-search:focus {
  border-color: #6366f1;
}
/* 窗口化模式：固定高度滚动容器 + 定高卡片（行高恒定才能计算滚动偏移） */
.playlist-scroll.windowed {
  max-height: calc(100vh - 320px);
  overflow-y: auto;
}
.playlist-grid.windowed .playlist-card {
  height: 104px;
  overflow: hidden;
  box-sizing: border-box;
}
.page-title {
  margin-bottom: 20px;
}
.page-title h2 {
  font-size: 22px;
  margin-bottom: 6px;
}
.hint {
  color: #94a3b8;
  font-size: 14px;
}
.playlist-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
}
.playlist-card {
  cursor: pointer;
  transition: background 0.15s;
}
.playlist-card:hover {
  background: rgba(148, 163, 184, 0.14);
}
.playlist-info {
  margin-bottom: 12px;
}
.playlist-name {
  font-size: 17px;
  font-weight: 500;
  margin-bottom: 6px;
}
.last-played-badge {
  display: inline-block;
  vertical-align: middle;
  margin-left: 8px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: normal;
  color: #34d399;
  border: 1px solid rgba(52, 211, 153, 0.5);
  border-radius: 6px;
}
.warn-bar {
  margin-bottom: 16px;
  padding: 10px 14px;
  font-size: 13px;
  color: #fbbf24;
  background: rgba(251, 191, 36, 0.08);
  border-radius: 8px;
}
.retry-link {
  color: #818cf8;
  cursor: pointer;
  text-decoration: underline;
  margin-left: 8px;
}
.playlist-count {
  font-size: 13px;
  color: #94a3b8;
}
.playlist-actions {
  display: flex;
  gap: 8px;
}
.playlist-actions .btn {
  padding: 6px 12px;
  font-size: 13px;
}
a {
  color: #6366f1;
  cursor: pointer;
  text-decoration: underline;
}
.modal {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}
.modal-content {
  width: 420px;
  max-width: calc(100% - 32px);
  display: flex;
  flex-direction: column;
}
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.modal-header h3 {
  font-size: 18px;
  font-weight: 500;
}
.modal-close {
  background: transparent;
  border: none;
  color: #94a3b8;
  font-size: 24px;
  line-height: 1;
  cursor: pointer;
  padding: 0 4px;
}
.modal-close:hover {
  color: #e2e8f0;
}
.modal-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-bottom: 20px;
}
.form-row {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.form-row label {
  font-size: 14px;
  color: #cbd5e1;
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
.confirm-text {
  font-size: 15px;
  line-height: 1.6;
}
.confirm-text strong {
  color: #e2e8f0;
}
.confirm-hint {
  font-size: 13px;
  color: #94a3b8;
  margin-top: 8px;
}
</style>
