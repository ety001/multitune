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
const editing = ref(null)
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

async function createPlaylist() {
  const name = newName.value.trim()
  if (!name) return
  await playlistStore.createPlaylist(props.id, name)
  newName.value = ''
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

function goPlayer(playlist) {
  router.push('/playlists/' + playlist.id)
}
</script>

<template>
  <div>
    <div class="page-header">
      <button class="btn btn-secondary" @click="router.push('/')">← 返回身份列表</button>
      <div class="page-title">
        <h2>{{ identity ? identity.name : '歌单管理' }}</h2>
        <p class="hint">选择歌单进入播放器，或在此管理该身份下的歌单。</p>
      </div>
    </div>

    <div v-if="error" class="error">{{ error }}</div>

    <div class="create-bar card">
      <input v-model="newName" type="text" placeholder="新歌单名称" />
      <button class="btn btn-primary" @click="createPlaylist">新建歌单</button>
    </div>

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
          <button class="btn btn-danger" @click="playlistStore.deletePlaylist(playlist.id)">删除</button>
        </div>
      </div>
    </div>

    <div v-if="editing" class="modal" @click.self="editing = null">
      <div class="modal-content card">
        <h3>编辑歌单</h3>
        <input v-model="editing.name" type="text" placeholder="歌单名称" />
        <div class="modal-actions">
          <button class="btn btn-secondary" @click="editing = null">取消</button>
          <button class="btn btn-primary" @click="saveEdit">保存</button>
        </div>
      </div>
    </div>
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
  margin-bottom: 6px;
}
.hint {
  color: #94a3b8;
  font-size: 14px;
}
.create-bar {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}
.create-bar input[type="text"] {
  flex: 1;
  min-width: 180px;
}
.playlist-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
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
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}
.modal-content {
  width: 360px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
