<script setup>
import { ref, computed, onMounted, watch } from 'vue'
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
const scanProgress = ref(null)

// 创建身份弹层
const showCreateIdentityModal = ref(false)
const newIdentityName = ref('')
const newIdentityColor = ref(randomColor())
const createIdentityLoading = ref(false)

// 创建歌单弹层
const showCreatePlaylistModal = ref(false)
const newPlaylistName = ref('')
const createPlaylistTargetIdentityId = ref('')
const createPlaylistLoading = ref(false)

function randomColor() {
  const colors = ['#ef4444', '#f97316', '#f59e0b', '#84cc16', '#10b981', '#06b6d4', '#0ea5e9', '#6366f1', '#8b5cf6', '#d946ef', '#f43f5e']
  return colors[Math.floor(Math.random() * colors.length)]
}

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

function openCreateIdentityModal() {
  newIdentityName.value = ''
  newIdentityColor.value = randomColor()
  showCreateIdentityModal.value = true
}

async function submitCreateIdentity() {
  const name = newIdentityName.value.trim()
  if (!name) {
    return
  }
  createIdentityLoading.value = true
  try {
    const identity = await identityStore.createIdentity(name, newIdentityColor.value)
    await identityStore.fetchIdentities()
    targetIdentityId.value = identity.id
    showCreateIdentityModal.value = false
  } catch (e) {
    alert('创建身份失败：' + e.message)
  } finally {
    createIdentityLoading.value = false
  }
}

function openCreatePlaylistModal() {
  newPlaylistName.value = ''
  createPlaylistTargetIdentityId.value = targetIdentityId.value
  showCreatePlaylistModal.value = true
}

async function submitCreatePlaylist() {
  const name = newPlaylistName.value.trim()
  const identityId = createPlaylistTargetIdentityId.value
  if (!name) {
    return
  }
  if (!identityId) {
    alert('请先选择目标身份')
    return
  }
  createPlaylistLoading.value = true
  try {
    const playlist = await playlistStore.createPlaylist(identityId, name)
    targetIdentityId.value = identityId
    await playlistStore.fetchPlaylists(identityId)
    targetPlaylistId.value = playlist.id
    showCreatePlaylistModal.value = false
  } catch (e) {
    alert('创建歌单失败：' + e.message)
  } finally {
    createPlaylistLoading.value = false
  }
}

function onIdentitySelectChange(e) {
  const value = e.target.value
  if (value === '__create_identity__') {
    targetIdentityId.value = ''
    openCreateIdentityModal()
    return
  }
  targetIdentityId.value = value
}

function onPlaylistSelectChange(e) {
  const value = e.target.value
  if (value === '__create_playlist__') {
    targetPlaylistId.value = ''
    openCreatePlaylistModal()
    return
  }
  targetPlaylistId.value = value
}

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

// 表头全选框：当前页所有条目都已勾选时为 true；
// 用户取消任意一项后自动变回未勾选
const allSelected = computed(() => {
  const items = fileStore.items
  if (!items || items.length === 0) return false
  for (const item of items) {
    if (selectedPaths.value.indexOf(item.path) < 0) return false
  }
  return true
})

function toggleSelectAll(e) {
  const checked = e.target.checked
  if (checked) {
    // 勾选当前页全部条目
    for (const item of fileStore.items) {
      if (selectedPaths.value.indexOf(item.path) < 0) {
        selectedPaths.value.push(item.path)
      }
    }
  } else {
    // 仅取消当前页条目，保留其他目录的勾选
    const pagePaths = new Set(fileStore.items.map((item) => item.path))
    selectedPaths.value = selectedPaths.value.filter((p) => !pagePaths.has(p))
  }
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

    <div>
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
        文件浏览器需要后端配置至少一个可访问的存储源才能浏览文件。
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
              <th class="select-cell">
                <label><input type="checkbox" :checked="allSelected" @change="toggleSelectAll" /></label>
              </th>
              <th>名称</th>
              <th style="width: 100px">类型</th>
              <th style="width: 100px">大小</th>
              <th style="width: 140px">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in fileStore.items" :key="item.path" :class="{ 'audio-row': item.is_audio }">
              <td class="select-cell">
                <label><input type="checkbox" :checked="isSelected(item.path)" @change="toggleSelect(item.path)" /></label>
              </td>
              <td>
                <span v-if="item.type === 'dir'" class="dir-name" @click="openDir(item.path)"><i class="fas fa-folder"></i> {{ item.name }}</span>
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

    <div class="target-panel card">
      <div class="target-row">
        <label>目标身份：</label>
        <select :value="targetIdentityId" @change="onIdentitySelectChange">
          <option value="">请选择</option>
          <option v-for="identity in identityStore.identities" :key="identity.id" :value="identity.id">
            {{ identity.name }}
          </option>
          <option value="__create_identity__">+ 创建身份</option>
        </select>
      </div>
      <div class="target-row">
        <label>目标歌单：</label>
        <select :value="targetPlaylistId" @change="onPlaylistSelectChange">
          <option value="">请选择</option>
          <option v-for="playlist in playlistStore.playlists" :key="playlist.id" :value="playlist.id">
            {{ playlist.name }}
          </option>
          <option value="__create_playlist__">+ 创建歌单</option>
        </select>
      </div>
      <div class="target-actions">
        <span v-if="addResult" class="add-result">{{ addResult }}</span>
        <button class="btn btn-primary" :disabled="fileStore.scanLoading" @click="scanAndAddSelected">
          {{ scanProgress ? `扫描中 ${scanProgress.current}/${scanProgress.total}` : '扫描并添加到歌单' }}
        </button>
      </div>
    </div>

    <!-- 创建身份弹层 -->
    <div v-if="showCreateIdentityModal" class="modal-overlay" @click.self="showCreateIdentityModal = false">
      <div class="modal-dialog-box">
        <div class="modal-header-box">
          <h3>创建身份</h3>
          <button class="modal-close" @click="showCreateIdentityModal = false">&times;</button>
        </div>
        <div class="modal-body-box">
          <div v-if="createIdentityLoading" class="modal-loading">创建中...</div>
          <div v-else>
            <div class="form-row">
              <label>身份名称</label>
              <input v-model="newIdentityName" type="text" placeholder="请输入身份名称" maxlength="50" />
            </div>
            <div class="form-row">
              <label>卡片颜色</label>
              <div class="color-row">
                <input v-model="newIdentityColor" type="color" />
                <button class="btn btn-secondary" @click="newIdentityColor = randomColor()">换个颜色</button>
              </div>
            </div>
          </div>
        </div>
        <div class="modal-footer-box">
          <button class="btn btn-secondary" :disabled="createIdentityLoading" @click="showCreateIdentityModal = false">取消</button>
          <button class="btn btn-primary" :disabled="createIdentityLoading || !newIdentityName.trim()" @click="submitCreateIdentity">创建</button>
        </div>
      </div>
    </div>

    <!-- 创建歌单弹层 -->
    <div v-if="showCreatePlaylistModal" class="modal-overlay" @click.self="showCreatePlaylistModal = false">
      <div class="modal-dialog-box">
        <div class="modal-header-box">
          <h3>创建歌单</h3>
          <button class="modal-close" @click="showCreatePlaylistModal = false">&times;</button>
        </div>
        <div class="modal-body-box">
          <div v-if="createPlaylistLoading" class="modal-loading">创建中...</div>
          <div v-else>
            <div class="form-row">
              <label>目标身份</label>
              <select v-model="createPlaylistTargetIdentityId">
                <option value="">请选择</option>
                <option v-for="identity in identityStore.identities" :key="identity.id" :value="identity.id">
                  {{ identity.name }}
                </option>
              </select>
            </div>
            <div class="form-row">
              <label>歌单名称</label>
              <input v-model="newPlaylistName" type="text" placeholder="请输入歌单名称" maxlength="50" />
            </div>
          </div>
        </div>
        <div class="modal-footer-box">
          <button class="btn btn-secondary" :disabled="createPlaylistLoading" @click="showCreatePlaylistModal = false">取消</button>
          <button class="btn btn-primary" :disabled="createPlaylistLoading || !newPlaylistName.trim() || !createPlaylistTargetIdentityId" @click="submitCreatePlaylist">创建</button>
        </div>
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
.file-table .select-cell {
  padding: 0;
  width: 48px;
}
.file-table .select-cell label {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 10px 12px;
  cursor: pointer;
}
.file-table .select-cell input[type='checkbox'] {
  width: 20px;
  height: 20px;
  cursor: pointer;
  accent-color: #6366f1;
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
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}
.modal-dialog-box {
  background: #0f172a;
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 16px;
  width: 100%;
  max-width: 420px;
  overflow: hidden;
}
.modal-header-box {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.15);
}
.modal-header-box h3 {
  font-size: 18px;
  margin: 0;
}
.modal-close {
  background: none;
  border: none;
  color: #94a3b8;
  font-size: 24px;
  cursor: pointer;
  line-height: 1;
}
.modal-close:hover {
  color: #e2e8f0;
}
.modal-body-box {
  padding: 20px;
}
.modal-footer-box {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid rgba(148, 163, 184, 0.15);
}
.modal-loading {
  text-align: center;
  padding: 40px 20px;
  color: #94a3b8;
}
.form-row {
  margin-bottom: 16px;
}
.form-row:last-child {
  margin-bottom: 0;
}
.form-row label {
  display: block;
  font-size: 14px;
  color: #94a3b8;
  margin-bottom: 8px;
}
.form-row input[type='text'],
.form-row select {
  width: 100%;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid rgba(148, 163, 184, 0.3);
  background: rgba(15, 23, 42, 0.5);
  color: #e2e8f0;
  font-size: 14px;
}
.form-row input[type='color'] {
  width: 60px;
  height: 40px;
  padding: 2px;
  border-radius: 8px;
  border: 1px solid rgba(148, 163, 184, 0.3);
  background: transparent;
  cursor: pointer;
}
.color-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
</style>
