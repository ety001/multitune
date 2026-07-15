import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fsApi, songApi } from '../api/client'

export const useFileBrowserStore = defineStore('fileBrowser', () => {
  const sources = ref([])
  const currentPath = ref('')
  const items = ref([])
  const parent = ref('')
  const loading = ref(false)
  const error = ref(null)
  const searchResults = ref([])
  const searchTotal = ref(0)
  const scanLoading = ref(false)
  const scanResult = ref(null)

  async function fetchSources() {
    loading.value = true
    error.value = null
    try {
      const data = await fsApi.sources()
      sources.value = data.items || []
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function listDirectory(path) {
    loading.value = true
    error.value = null
    try {
      const data = await fsApi.list(path)
      currentPath.value = data.path
      parent.value = data.parent
      items.value = data.items || []
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function scanPath(path) {
    scanLoading.value = true
    error.value = null
    try {
      const data = await fsApi.scan(path)
      scanResult.value = data
      return data
    } catch (e) {
      error.value = e.message
      throw e
    } finally {
      scanLoading.value = false
    }
  }

  async function searchSongs(q, source, limit = 50, offset = 0) {
    loading.value = true
    error.value = null
    try {
      const data = await fsApi.search({ q, source, limit, offset })
      searchResults.value = data.items || []
      searchTotal.value = data.total || 0
      return data
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function listSongs(params = {}) {
    loading.value = true
    error.value = null
    try {
      const data = await songApi.list(params)
      searchResults.value = data.items || []
      searchTotal.value = data.total || 0
      return data
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  function resetSearch() {
    searchResults.value = []
    searchTotal.value = 0
  }

  return {
    sources,
    currentPath,
    items,
    parent,
    loading,
    error,
    searchResults,
    searchTotal,
    scanLoading,
    scanResult,
    fetchSources,
    listDirectory,
    scanPath,
    searchSongs,
    listSongs,
    resetSearch,
  }
})
