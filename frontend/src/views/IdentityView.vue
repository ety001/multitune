<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useIdentityStore } from '../stores/identity'

const router = useRouter()
const store = useIdentityStore()

const newName = ref('')
const newColor = ref('#6366f1')
const showCreateModal = ref(false)
const editing = ref(null)
const deleting = ref(null)
const colorInputRef = ref(null)
const editColorInputRef = ref(null)

onMounted(() => {
  store.fetchIdentities()
})

function randomColor() {
  const colors = [
    '#ef4444', '#f97316', '#f59e0b', '#84cc16', '#22c55e',
    '#10b981', '#14b8a6', '#06b6d4', '#0ea5e9', '#3b82f6',
    '#6366f1', '#8b5cf6', '#a855f7', '#d946ef', '#ec4899',
    '#f43f5e', '#78716c', '#475569', '#64748b', '#94a3b8',
  ]
  return colors[Math.floor(Math.random() * colors.length)]
}

function openCreateModal() {
  newName.value = ''
  newColor.value = randomColor()
  showCreateModal.value = true
}

function closeCreateModal() {
  showCreateModal.value = false
}

async function createIdentity() {
  const name = newName.value.trim()
  if (!name) return
  await store.createIdentity(name, newColor.value)
  closeCreateModal()
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

function startDelete(identity) {
  deleting.value = { ...identity }
}

function closeDeleteModal() {
  deleting.value = null
}

async function confirmDelete() {
  if (!deleting.value) return
  await store.deleteIdentity(deleting.value.id)
  closeDeleteModal()
}

function goPlaylists(id) {
  router.push('/identities/' + id + '/playlists')
}
</script>

<template>
  <div>
    <div class="page-header">
      <div class="page-title">
        <h2>身份管理</h2>
        <p class="hint">点击身份卡片进入对应歌单，快速切换默认上车身份。</p>
      </div>
      <button class="btn btn-primary" @click="openCreateModal">+ 新建身份</button>
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
          <button class="btn btn-danger" @click="startDelete(identity)">删除</button>
        </div>
      </div>
    </div>

    <!-- 新建身份弹层 -->
    <div v-if="showCreateModal" class="modal">
      <div class="modal-content card">
        <div class="modal-header">
          <h3>新建身份</h3>
          <button class="modal-close" @click="closeCreateModal">&times;</button>
        </div>
        <div class="modal-body">
          <div class="form-row">
            <label>身份名称</label>
            <input v-model="newName" type="text" placeholder="例如：爸爸" @keyup.enter="createIdentity" />
          </div>
          <div class="form-row">
            <label>卡片颜色</label>
            <div class="color-picker-row">
              <div class="color-picker" @click="colorInputRef?.value?.click()">
                <div class="color-swatch" :style="{ background: newColor }"></div>
                <span class="color-value">{{ newColor }}</span>
                <input ref="colorInputRef" v-model="newColor" type="color" class="color-input" />
              </div>
              <button class="btn btn-secondary btn-small" @click="newColor = randomColor()">换个颜色</button>
            </div>
          </div>
        </div>
        <div class="modal-actions">
          <button class="btn btn-secondary" @click="closeCreateModal">取消</button>
          <button class="btn btn-primary" :disabled="!newName.trim()" @click="createIdentity">创建</button>
        </div>
      </div>
    </div>

    <!-- 删除确认弹层 -->
    <div v-if="deleting" class="modal">
      <div class="modal-content card">
        <div class="modal-header">
          <h3>删除身份</h3>
          <button class="modal-close" @click="closeDeleteModal">&times;</button>
        </div>
        <div class="modal-body">
          <p class="confirm-text">
            确定要删除身份 <strong>{{ deleting.name }}</strong> 吗？
          </p>
          <p class="confirm-hint">删除后该身份下的所有歌单和播放记录将无法恢复。</p>
        </div>
        <div class="modal-actions">
          <button class="btn btn-secondary" @click="closeDeleteModal">取消</button>
          <button class="btn btn-danger" @click="confirmDelete">确认删除</button>
        </div>
      </div>
    </div>

    <!-- 编辑身份弹层 -->
    <div v-if="editing" class="modal">
      <div class="modal-content card">
        <div class="modal-header">
          <h3>编辑身份</h3>
          <button class="modal-close" @click="editing = null">&times;</button>
        </div>
        <div class="modal-body">
          <div class="form-row">
            <label>身份名称</label>
            <input v-model="editing.name" type="text" placeholder="身份名称" @keyup.enter="saveEdit" />
          </div>
          <div class="form-row">
            <label>卡片颜色</label>
            <div class="color-picker-row">
              <div class="color-picker" @click="editColorInputRef?.value?.click()">
                <div class="color-swatch" :style="{ background: editing.color }"></div>
                <span class="color-value">{{ editing.color }}</span>
                <input :ref="editColorInputRef" v-model="editing.color" type="color" class="color-input" />
              </div>
              <button class="btn btn-secondary btn-small" @click="editing.color = randomColor()">换个颜色</button>
            </div>
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
.page-title h2 {
  font-size: 22px;
  margin-bottom: 6px;
}
.hint {
  color: #94a3b8;
  font-size: 14px;
}
.identity-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
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
.color-picker-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.color-picker {
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
}
.color-swatch {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  border: 2px solid rgba(148, 163, 184, 0.3);
  flex-shrink: 0;
}
.color-value {
  font-size: 14px;
  color: #94a3b8;
  font-family: monospace;
}
.color-input {
  position: absolute;
  opacity: 0;
  width: 0;
  height: 0;
  pointer-events: none;
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
