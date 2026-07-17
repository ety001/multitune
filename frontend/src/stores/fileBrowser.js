import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fsApi } from '../api/client'

export const useFileBrowserStore = defineStore('fileBrowser', () => {
  const sources = ref([])
  const currentPath = ref('')
  const items = ref([])
  const parent = ref('')
  const loading = ref(false)
  const error = ref(null)
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

  return {
    sources,
    currentPath,
    items,
    parent,
    loading,
    error,
    scanLoading,
    scanResult,
    fetchSources,
    listDirectory,
    scanPath,
  }
})
