<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useIdentityStore } from '../stores/identity'

const router = useRouter()
const store = useIdentityStore()

const newName = ref('')
const newColor = ref('#6366f1')
const editing = ref(null)

onMounted(() => {
  store.fetchIdentities()
})

async function createIdentity() {
  const name = newName.value.trim()
  if (!name) return
  await store.createIdentity(name, newColor.value)
  newName.value = ''
  newColor.value = '#6366f1'
}

function startEdit(identity) {
  editing.value = {
    id: identity.id,
    name: identity.name,
    color: identity.avatar_color,
  }
}

async function saveEdit() {
  if (!editing.value) return
  const name = editing.value.name.trim()
  if (!name) return
  await store.updateIdentity(editing.value.id, {
    name,
    avatar_color: editing.value.color,
  })
  editing.value = null
}

function goPlaylists(id) {
  router.push('/identities/' + id + '/playlists')
}
</script>

<template>
  <div>
    <div class="page-title">
      <h2>身份管理</h2>
      <p class="hint">点击身份卡片进入对应歌单，快速切换默认上车身份。</p>
    </div>

    <div class="create-bar card">
      <input v-model="newName" type="text" placeholder="新身份名称，例如：爸爸" />
      <input v-model="newColor" type="color" title="卡片颜色" />
      <button class="btn btn-primary" @click="createIdentity">新建身份</button>
    </div>

    <div v-if="store.error" class="error">{{ store.error }}</div>

    <div v-if="store.loading" class="empty">加载中...</div>
    <div v-else-if="store.identities.length === 0" class="empty">暂无身份，请先创建一个身份。</div>

    <div v-else class="identity-grid">
      <div
        v-for="identity in store.identities"
        :key="identity.id"
        class="identity-card card"
        :style="{ borderLeftColor: identity.avatar_color }"
        @click="goPlaylists(identity.id)"
      >
        <div class="identity-header">
          <div class="identity-avatar" :style="{ background: identity.avatar_color }"></div>
          <div class="identity-meta">
            <div class="identity-name">{{ identity.name }}</div>
            <div v-if="identity.is_default" class="default-badge">默认身份</div>
          </div>
        </div>
        <div class="identity-actions" @click.stop>
          <button v-if="!identity.is_default" class="btn btn-secondary" @click="store.setDefault(identity.id)">
            设为默认
          </button>
          <button class="btn btn-secondary" @click="startEdit(identity)">编辑</button>
          <button class="btn btn-danger" @click="store.deleteIdentity(identity.id)">删除</button>
        </div>
      </div>
    </div>

    <div v-if="editing" class="modal" @click.self="editing = null">
      <div class="modal-content card">
        <h3>编辑身份</h3>
        <input v-model="editing.name" type="text" placeholder="身份名称" />
        <input v-model="editing.color" type="color" />
        <div class="modal-actions">
          <button class="btn btn-secondary" @click="editing = null">取消</button>
          <button class="btn btn-primary" @click="saveEdit">保存</button>
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
.identity-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 16px;
}
.identity-card {
  border-left: 4px solid;
  cursor: pointer;
  transition: transform 0.15s, background 0.15s;
}
.identity-card:hover {
  background: rgba(148, 163, 184, 0.14);
}
.identity-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}
.identity-avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  flex-shrink: 0;
}
.identity-name {
  font-size: 18px;
  font-weight: 500;
}
.default-badge {
  display: inline-block;
  margin-top: 4px;
  font-size: 12px;
  color: #10b981;
}
.identity-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.identity-actions .btn {
  padding: 6px 12px;
  font-size: 13px;
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
