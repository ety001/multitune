<script setup>
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { onBeforeUnmount, onMounted } from 'vue'
import { usePlayerStore } from './stores/player'

const route = useRoute()
const router = useRouter()
const playerStore = usePlayerStore()

function saveBeforeLeave() {
  playerStore.savePlaybackState()
}

onMounted(() => {
  window.addEventListener('beforeunload', saveBeforeLeave)
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'hidden') {
      playerStore.savePlaybackState()
    }
  })
})

onBeforeUnmount(() => {
  window.removeEventListener('beforeunload', saveBeforeLeave)
})

function goToPlayer() {
  if (playerStore.currentPlaylist) {
    router.push('/playlists/' + playerStore.currentPlaylist.id)
  }
}
</script>

<template>
  <div class="app">
    <header class="app-header">
      <div class="app-header-inner">
        <div class="app-brand">
          <h1>多音盒 MultiTune</h1>
          <span class="app-tag">完整版</span>
        </div>
        <nav class="app-nav">
          <RouterLink to="/identities" :class="{ active: route.path.startsWith('/identities') }">身份</RouterLink>
          <RouterLink to="/file-browser" :class="{ active: route.path === '/file-browser' }">文件浏览器</RouterLink>
        </nav>
      </div>
    </header>

    <main class="app-main">
      <RouterView />
    </main>

    <div v-if="playerStore.currentSong" class="mini-player">
      <div class="mini-info" @click="goToPlayer">
        <div class="mini-title">{{ playerStore.currentSong.title }}</div>
        <div class="mini-artist">{{ playerStore.currentSong.artist || '' }}</div>
      </div>
      <div class="mini-controls">
        <button class="btn btn-secondary btn-small" @click="playerStore.prev">⏮</button>
        <button class="btn btn-primary btn-small" @click="playerStore.togglePlay">
          {{ playerStore.isPlaying ? '⏸' : '▶' }}
        </button>
        <button class="btn btn-secondary btn-small" @click="playerStore.next">⏭</button>
      </div>
      <div class="mini-progress">
        <div
          class="mini-progress-fill"
          :style="{ width: (playerStore.duration ? (playerStore.currentTime / playerStore.duration) * 100 : 0) + '%' }"
        ></div>
      </div>
    </div>
  </div>
</template>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  background: #0f172a;
  color: #e2e8f0;
}

.app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.app-header {
  background: rgba(15, 23, 42, 0.95);
  border-bottom: 1px solid rgba(148, 163, 184, 0.15);
}

.app-header-inner {
  max-width: 1200px;
  margin: 0 auto;
  width: 100%;
  padding: 16px 24px;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 12px;
}

.app-brand {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.app-brand h1 {
  font-size: 20px;
}

.app-tag {
  font-size: 12px;
  padding: 2px 8px;
  background: #6366f1;
  color: #fff;
  border-radius: 4px;
}

.app-nav {
  display: flex;
  gap: 16px;
}

.app-nav a {
  color: #94a3b8;
  text-decoration: none;
  font-size: 14px;
  padding: 4px 0;
  border-bottom: 2px solid transparent;
}

.app-nav a:hover,
.app-nav a.active {
  color: #e2e8f0;
  border-bottom-color: #6366f1;
}

.app-main {
  flex: 1;
  padding: 24px;
  padding-bottom: 100px;
  max-width: 1200px;
  margin: 0 auto;
  width: 100%;
}

.card {
  background: rgba(148, 163, 184, 0.08);
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 16px;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 10px 18px;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  transition: opacity 0.2s;
}

.btn:hover {
  opacity: 0.9;
}

.btn-primary {
  background: #6366f1;
  color: #fff;
}

.btn-danger {
  background: #ef4444;
  color: #fff;
}

.btn-secondary {
  background: rgba(148, 163, 184, 0.15);
  color: #cbd5e1;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

input[type="text"],
input[type="color"] {
  padding: 10px 12px;
  border: 1px solid rgba(148, 163, 184, 0.3);
  border-radius: 8px;
  background: rgba(15, 23, 42, 0.5);
  color: #e2e8f0;
  font-size: 14px;
}

input[type="text"]:focus {
  outline: none;
  border-color: #6366f1;
}

.empty {
  padding: 40px;
  text-align: center;
  color: #94a3b8;
}

.error {
  padding: 12px 16px;
  background: rgba(239, 68, 68, 0.1);
  color: #fca5a5;
  border-radius: 8px;
  margin-bottom: 16px;
}

.mini-player {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  background: rgba(15, 23, 42, 0.98);
  border-top: 1px solid rgba(148, 163, 184, 0.15);
  padding: 12px 24px;
  display: flex;
  align-items: center;
  gap: 16px;
  z-index: 50;
}

.mini-info {
  flex: 1;
  min-width: 120px;
  cursor: pointer;
}

.mini-title {
  font-size: 14px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mini-artist {
  font-size: 12px;
  color: #94a3b8;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mini-controls {
  display: flex;
  gap: 8px;
}

.mini-progress {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: rgba(148, 163, 184, 0.2);
}

.mini-progress-fill {
  height: 100%;
  background: #6366f1;
}

@media (max-width: 600px) {
  .app-header-inner {
    padding: 12px 16px;
  }
  .app-nav {
    width: 100%;
  }
  .app-main {
    padding: 16px;
    padding-bottom: 90px;
  }
  .mini-player {
    padding: 10px 16px;
  }
}
</style>
