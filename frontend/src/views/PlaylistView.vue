<script setup>
import { ref, onMounted } from 'vue'
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
const newName = ref('')
const showCreateModal = ref(false)
const editing = ref(null)
const deleting = ref(null)
const error = ref(null)

onMounted(async () => {
  await loadIdentity()
  await playlistStore.fetchPlaylists(props.id)
})

async function loadIdentity() {
  try {
    identity.value = await identityApi.get(props.id)
    playerStore.setCurrentIdentity(identity.value)
  } catch (e) {
    error.value = e.message
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
}

function goPlayer(playlist) {
  router.push('/playlists/' + playlist.id)
}
</script>

<template>
  <div>
    <div class="page-header">
      <button class="btn btn-secondary" @click="router.push('/')">&larr; 返回身份列表</button>
      <button class="btn btn-primary" @click="openCreateModal">+ 新建歌单</button>
    </div>

    <div class="page-title">
      <h2>{{ identity ? identity.name : '歌单管理' }}</h2>
      <p class="hint">选择歌单进入播放器，或在此管理该身份下的歌单。</p>
    </div>

    <div v-if="error" class="error">{{ error }}</div>

    <div v-if="playlistStore.loading" class="empty">加载中...</div>
    <div v-else-if="playlistStore.playlists.length === 0" class="empty">
      该身份下还没有歌单，去<a @click="router.push('/file-browser')">文件浏览器</a>添加歌曲吧。
    </div>

    <div v-else class="playlist-grid">
      <div v-for="playlist in playlistStore.playlists" :key="playlist.id" class="playlist-card card">
        <div class="playlist-info" @click="goPlayer(playlist)">
          <div class="playlist-name">{{ playlist.name }}</div>
          <div class="playlist-count">{{ playlist.song_count || 0 }} 首歌曲</div>
        </div>
        <div class="playlist-actions" @click.stop>
          <button class="btn btn-secondary" @click="startEdit(playlist)">编辑</button>
          <button class="btn btn-danger" @click="startDelete(playlist)">删除</button>
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
