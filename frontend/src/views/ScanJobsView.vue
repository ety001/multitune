<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { scanApi } from '../api/client'

const jobs = ref([])
const loading = ref(false)
const error = ref(null)
let pollTimer = null

function formatTime(unix) {
  if (!unix) return '-'
  const d = new Date(unix * 1000)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function isRunning(job) {
  return job.status === 'pending' || job.status === 'counting' || job.status === 'scanning'
}

const statusText = {
  pending: '等待中',
  counting: '统计中',
  scanning: '扫描中',
  done: '完成',
  error: '失败',
}

const statusClass = {
  pending: 'status-pending',
  counting: 'status-running',
  scanning: 'status-running',
  done: 'status-done',
  error: 'status-error',
}

async function fetchJobs() {
  try {
    const data = await scanApi.list(50)
    jobs.value = data.items || []
    error.value = null
    schedulePoll()
  } catch (e) {
    error.value = e.message
  }
}

// 有进行中的任务时每 2 秒轮询，全部结束后停止
function schedulePoll() {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
  if (jobs.value.some(isRunning)) {
    pollTimer = setTimeout(async () => {
      await fetchJobs()
    }, 2000)
  }
}

function refresh() {
  loading.value = true
  fetchJobs().finally(() => {
    loading.value = false
  })
}

onMounted(refresh)
onUnmounted(() => {
  if (pollTimer) clearTimeout(pollTimer)
})
</script>

<template>
  <div>
    <div class="page-title">
      <h2>扫描任务</h2>
      <p class="hint">扫描在服务端后台执行，关闭页面不影响歌曲入库与加入歌单；可在此回看每个任务的结果。</p>
    </div>

    <div class="toolbar">
      <button class="btn btn-secondary" :disabled="loading" @click="refresh">
        <i class="fas fa-sync-alt"></i> 刷新
      </button>
    </div>

    <div v-if="error" class="error">{{ error }}</div>
    <div v-if="loading && jobs.length === 0" class="empty card">加载中...</div>

    <table v-else-if="jobs.length > 0" class="job-table card">
      <thead>
        <tr>
          <th>创建时间</th>
          <th>目标歌单</th>
          <th>扫描路径</th>
          <th>状态</th>
          <th>进度/结果</th>
          <th>说明</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="job in jobs" :key="job.id">
          <td>{{ formatTime(job.created_at) }}</td>
          <td>{{ job.playlist_name || job.playlist_id }}</td>
          <td class="paths">
            <div v-for="p in job.paths" :key="p" class="path-item" :title="p">{{ p }}</div>
          </td>
          <td><span :class="statusClass[job.status]">{{ statusText[job.status] || job.status }}</span></td>
          <td>
            <template v-if="job.status === 'scanning'">{{ job.current }}/{{ job.total }}</template>
            <template v-else-if="job.status === 'done'">新增 {{ job.added }} 首<span v-if="job.updated">，更新 {{ job.updated }} 首</span></template>
            <template v-else>-</template>
          </td>
          <td class="message">{{ job.message || '-' }}</td>
        </tr>
      </tbody>
    </table>

    <div v-else class="empty card">还没有扫描任务，去文件浏览器勾选目录「扫描并添加到歌单」吧。</div>
  </div>
</template>

<style scoped>
.toolbar {
  margin-bottom: 16px;
  display: flex;
  gap: 8px;
}
.job-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}
.job-table th,
.job-table td {
  padding: 10px 12px;
  text-align: left;
  border-bottom: 1px solid var(--border-color, #eee);
  vertical-align: top;
}
.job-table th {
  font-weight: 600;
  color: var(--text-secondary, #666);
}
.paths {
  max-width: 320px;
}
.path-item {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.message {
  max-width: 260px;
  word-break: break-all;
}
.status-pending {
  color: #999;
}
.status-running {
  color: #1976d2;
}
.status-done {
  color: #2e7d32;
}
.status-error {
  color: #c62828;
}
</style>
