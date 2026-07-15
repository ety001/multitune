import { defineStore } from 'pinia'
import { ref } from 'vue'
import { playlistApi } from '../api/client'

export const usePlaylistStore = defineStore('playlist', () => {
  const playlists = ref([])
  const currentPlaylist = ref(null)
  const loading = ref(false)
  const error = ref(null)

  async function fetchPlaylists(identityId) {
    loading.value = true
    error.value = null
    try {
      const data = await playlistApi.listByIdentity(identityId)
      playlists.value = data.items || []
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function createPlaylist(identityId, name, sortOrder = 0) {
    const data = await playlistApi.create(identityId, { name, sort_order: sortOrder })
    playlists.value.push(data)
    return data
  }

  async function updatePlaylist(id, fields) {
    const payload = {}
    if (fields.name !== undefined) payload.name = fields.name
    if (fields.cover_url !== undefined) payload.cover_url = fields.cover_url
    if (fields.sort_order !== undefined) payload.sort_order = fields.sort_order

    const data = await playlistApi.update(id, payload)
    const idx = playlists.value.findIndex((item) => item.id === id)
    if (idx !== -1) {
      playlists.value[idx] = data
    }
    if (currentPlaylist.value && currentPlaylist.value.id === id) {
      currentPlaylist.value = { ...currentPlaylist.value, ...data }
    }
    return data
  }

  async function deletePlaylist(id) {
    await playlistApi.delete(id)
    playlists.value = playlists.value.filter((item) => item.id !== id)
    if (currentPlaylist.value && currentPlaylist.value.id === id) {
      currentPlaylist.value = null
    }
  }

  async function fetchPlaylistDetail(id, limit = 200, offset = 0) {
    loading.value = true
    error.value = null
    try {
      const data = await playlistApi.get(id, { limit, offset })
      currentPlaylist.value = data
      return data
    } catch (e) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  async function addSongs(playlistId, songIds) {
    const data = await playlistApi.addSongs(playlistId, songIds)
    return data
  }

  async function removeSong(playlistId, songId) {
    await playlistApi.removeSong(playlistId, songId)
    if (currentPlaylist.value && currentPlaylist.value.id === playlistId) {
      currentPlaylist.value.songs = currentPlaylist.value.songs.filter((s) => s.id !== songId)
      currentPlaylist.value.song_count = Math.max(0, currentPlaylist.value.song_count - 1)
    }
  }

  async function reorderSongs(playlistId, songIds) {
    await playlistApi.updateOrder(playlistId, songIds)
    if (currentPlaylist.value && currentPlaylist.value.id === playlistId) {
      const map = new Map(currentPlaylist.value.songs.map((s) => [s.id, s]))
      currentPlaylist.value.songs = songIds.map((id) => map.get(id)).filter(Boolean)
    }
  }

  return {
    playlists,
    currentPlaylist,
    loading,
    error,
    fetchPlaylists,
    createPlaylist,
    updatePlaylist,
    deletePlaylist,
    fetchPlaylistDetail,
    addSongs,
    removeSong,
    reorderSongs,
  }
})
