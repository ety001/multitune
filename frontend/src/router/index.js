import { createRouter, createWebHashHistory } from 'vue-router'
import IdentityView from '../views/IdentityView.vue'
import PlaylistView from '../views/PlaylistView.vue'
import FileBrowserView from '../views/FileBrowserView.vue'
import PlayerView from '../views/PlayerView.vue'

const routes = [
  { path: '/', redirect: '/identities' },
  { path: '/identities', component: IdentityView },
  { path: '/identities/:id/playlists', component: PlaylistView, props: true },
  { path: '/playlists/:id', component: PlayerView, props: true },
  { path: '/file-browser', component: FileBrowserView },
]

export default createRouter({
  history: createWebHashHistory(),
  routes,
})
