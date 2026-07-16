<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useFileBrowserStore } from '../stores/fileBrowser'
import { useIdentityStore } from '../stores/identity'
import { usePlaylistStore } from '../stores/playlist'
import { usePlayerStore } from '../stores/player'
import { scanApi } from '../api/client'

const router = useRouter()
const fileStore = useFileBrowserStore()
const identityStore = useIdentityStore()
const playlistStore = usePlaylistStore()
const playerStore = usePlayerStore()

const selectedPaths = ref([])
const targetIdentityId = ref('')
const targetPlaylistId = ref('')
const addResult = ref('')
const searchQuery = ref('')
const viewMode = ref('browse')
const scanProgress = ref(null)

onMounted(async () => {
  await identityStore.fetchIdentities()
  await fileStore.fetchSources()
  if (fileStore.sources.length > 0) {
    await fileStore.listDirectory(fileStore.sources[0].path)
  }
})

watch(targetIdentityId, async (id) => {
  targetPlaylistId.value = ''
  if (id) {
    await playlistStore.fetchPlaylists(id)
  } else {
    playlistStore.playlists = []
  }
})

function toggleSelect(path) {
  const idx = selectedPaths.value.indexOf(path)
  if (idx >= 0) {
    selectedPaths.value.splice(idx, 1)
  } else {
    selectedPaths.value.push(path)
  }
}

function isSelected(path) {
  return selectedPaths.value.indexOf(path) >= 0
}

async function openDir(path) {
  selectedPaths.value = []
  await fileStore.listDirectory(path)
}

async function goUp() {
  if (fileStore.parent && fileStore.parent !== fileStore.currentPath) {
    await openDir(fileStore.parent)
  }
}

function enterSource(path) {
  openDir(path)
}

async function scanAndAddSelected() {
  addResult.value = ''
  if (!targetPlaylistId.value) {
    addResult.value = '请先选择目标歌单'
    return
  }
  if (selectedPaths.value.length === 0) {
    addResult.value = '请先勾选要添加的文件或文件夹'
    return
  }

  fileStore.scanLoading = true
  scanProgress.value = { current: 0, total: 0 }

  try {
    const job = await scanApi.createJob({
      paths: selectedPaths.value,
      playlist_id: targetPlaylistId.value,
    })

    const jobId = job.id
    let finished = false
    while (!finished) {
      const status = await scanApi.getJob(jobId)
      scanProgress.value = { current: status.current, total: status.total }

      if (status.status === 'done') {
        addResult.value = `成功添加 ${status.added || 0} 首歌曲`
        selectedPaths.value = []
        finished = true
      } else if (status.status === 'error') {
        addResult.value = '扫描失败：' + (status.message || '未知错误')
        finished = true
      } else {
        await new Promise((resolve) => setTimeout(resolve, 500))
      }
    }
  } catch (e) {
    addResult.value = '扫描失败：' + e.message
  } finally {
    fileStore.scanLoading = false
    scanProgress.value = null
  }
}

async function playFile(path) {
  try {
    const result = await fileStore.scanPath(path)
    if (result && result.songs && result.songs.length > 0) {
      playerStore.playSong(result.songs[0])
    }
  } catch (e) {
    fileStore.error = e.message
  }
}

async function doSearch() {
  if (!searchQuery.value.trim()) {
    viewMode.value = 'browse'
    return
  }
  viewMode.value = 'search'
  await fileStore.searchSongs(searchQuery.value.trim(), '', 50, 0)
}

function addSearchSong(song) {
  if (!targetPlaylistId.value) {
    addResult.value = '请先选择目标歌单'
    return
  }
  playlistStore.addSongs(targetPlaylistId.value, [song.id])
    .then((data) => {
      addResult.value = `已添加 1 首歌曲`
    })
    .catch((e) => {
      addResult.value = '添加失败：' + e.message
    })
}

function formatBytes(bytes) {
  if (!bytes) return ''
  const kb = bytes / 1024
  if (kb < 1024) return kb.toFixed(1) + ' KB'
  return (kb / 1024).toFixed(1) + ' MB'
}
</script>

<template>
  <div>
    <div class="page-title">
      <h2>文件浏览器</h2>
      <p class="hint">浏览存储源，勾选音频文件或文件夹后添加到指定歌单。</p>
    </div>

    <div class="search-bar card">
      <input v-model="searchQuery" type="text" placeholder="搜索已扫描的歌曲..." @keyup.enter="doSearch" />
      <button class="btn btn-secondary" @click="doSearch">搜索</button>
      <button v-if="viewMode === 'search'" class="btn btn-secondary" @click="viewMode = 'browse'">返回浏览</button>
    </div>

    <div v-if="viewMode === 'browse'">
      <div class="sources card">
        <div class="sources-label">存储源：</div>
        <button
          v-for="source in fileStore.sources"
          :key="source.id"
          class="btn"
          :class="source.available ? 'btn-secondary' : 'btn-disabled'"
          :disabled="!source.available"
          @click="enterSource(source.path)"
        >
          {{ source.name }}
          <span v-if="!source.available" class="unavailable">(不可用)</span>
        </button>
        <span v-if="!fileStore.loading && fileStore.sources.length === 0" class="no-sources">
          未检测到根目录访问权限，请检查容器挂载与运行权限。
        </span>
      </div>

      <div v-if="!fileStore.loading && fileStore.sources.length === 0" class="empty card">
        文件浏览器需要后端能够访问容器文件系统根目录，请检查挂载与权限。
      </div>

      <div v-if="fileStore.sources.length > 0">
        <div class="breadcrumb card">
          <button class="btn btn-secondary" :disabled="!fileStore.parent || fileStore.parent === fileStore.currentPath" @click="goUp">
            ↑ 上级
          </button>
          <span class="path">{{ fileStore.currentPath || '/' }}</span>
        </div>

        <div v-if="fileStore.error" class="error">{{ fileStore.error }}</div>
        <div v-if="fileStore.loading" class="empty">加载中...</div>

        <table v-else-if="fileStore.items.length > 0" class="file-table card">
          <thead>
            <tr>
              <th style="width: 40px">选择</th>
              <th>名称</th>
              <th style="width: 100px">类型</th>
              <th style="width: 100px">大小</th>
              <th style="width: 140px">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in fileStore.items" :key="item.path" :class="{ 'audio-row': item.is_audio }">
              <td>
                <input type="checkbox" :checked="isSelected(item.path)" @change="toggleSelect(item.path)" />
              </td>
              <td>
                <span v-if="item.type === 'dir'" class="dir-name" @click="openDir(item.path)">📁 {{ item.name }}</span>
                <span v-else>{{ item.name }}</span>
              </td>
              <td>{{ item.type === 'dir' ? '文件夹' : item.is_audio ? '音频' : '文件' }}</td>
              <td>{{ formatBytes(item.size) }}</td>
              <td>
                <button v-if="item.type === 'dir'" class="btn btn-secondary btn-small" @click="fileStore.scanPath(item.path)">扫描</button>
                <button v-if="item.is_audio" class="btn btn-secondary btn-small" @click="playFile(item.path)">播放</button>
              </td>
            </tr>
          </tbody>
        </table>

        <div v-else class="empty card">当前目录为空</div>
      </div>
    </div>

    <div v-else-if="viewMode === 'search'">
      <div v-if="fileStore.loading" class="empty">搜索中...</div>
      <div v-else-if="fileStore.searchResults.length === 0" class="empty card">未找到匹配的歌曲</div>
      <table v-else class="file-table card">
        <thead>
          <tr>
            <th>歌曲</th>
            <th>艺术家</th>
            <th>专辑</th>
            <th>来源</th>
            <th style="width: 100px">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="song in fileStore.searchResults" :key="song.id">
            <td>{{ song.title }}</td>
            <td>{{ song.artist || '-' }}</td>
            <td>{{ song.album || '-' }}</td>
            <td>{{ song.source }}</td>
            <td>
              <button class="btn btn-secondary btn-small" @click="playerStore.playSong(song)">播放</button>
              <button class="btn btn-primary btn-small" @click="addSearchSong(song)">添加</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="target-panel card">
      <div class="target-row">
        <label>目标身份：</label>
        <select v-model="targetIdentityId">
          <option value="">请选择</option>
          <option v-for="identity in identityStore.identities" :key="identity.id" :value="identity.id">
            {{ identity.name }}
          </option>
        </select>
      </div>
      <div class="target-row">
        <label>目标歌单：</label>
        <select v-model="targetPlaylistId">
          <option value="">请选择</option>
          <option v-for="playlist in playlistStore.playlists" :key="playlist.id" :value="playlist.id">
            {{ playlist.name }}
          </option>
        </select>
      </div>
      <div class="target-actions">
        <span v-if="addResult" class="add-result">{{ addResult }}</span>
        <button class="btn btn-primary" :disabled="fileStore.scanLoading" @click="scanAndAddSelected">
          {{ scanProgress ? `扫描中 ${scanProgress.current}/${scanProgress.total}` : '扫描并添加到歌单' }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
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
.search-bar {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}
.search-bar input {
  flex: 1;
  min-width: 180px;
}
.sources {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.sources-label {
  font-weight: 500;
}
.unavailable {
  color: #94a3b8;
  font-size: 12px;
}
.no-sources {
  color: #fca5a5;
  font-size: 14px;
}
.breadcrumb {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.path {
  color: #cbd5e1;
  font-size: 14px;
  word-break: break-all;
}
.file-table {
  width: 100%;
  border-collapse: collapse;
}
.file-table th,
.file-table td {
  text-align: left;
  padding: 10px 12px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.1);
}
.file-table th {
  color: #94a3b8;
  font-weight: 500;
  font-size: 13px;
}
.dir-name {
  color: #6366f1;
  cursor: pointer;
}
.dir-name:hover {
  text-decoration: underline;
}
.audio-row {
  background: rgba(99, 102, 241, 0.06);
}
.btn-small {
  padding: 4px 10px;
  font-size: 12px;
  margin-right: 6px;
}
.btn-disabled {
  background: rgba(148, 163, 184, 0.08);
  color: #64748b;
  cursor: not-allowed;
}
.target-panel {
  position: sticky;
  bottom: 0;
  margin-top: 20px;
  display: flex;
  gap: 16px;
  align-items: center;
  flex-wrap: wrap;
  background: #0f172a;
  border-top: 1px solid rgba(148, 163, 184, 0.15);
  z-index: 10;
}
.target-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.target-row select {
  padding: 8px 10px;
  border-radius: 8px;
  border: 1px solid rgba(148, 163, 184, 0.3);
  background: rgba(15, 23, 42, 0.5);
  color: #e2e8f0;
  min-width: 140px;
}
.target-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 12px;
}
.add-result {
  font-size: 13px;
  color: #10b981;
}
</style>
