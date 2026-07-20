(function($) {
  'use strict';

  window.MultiTunePlayer = {
    options: null,
    playlist: null,
    songIds: [],          // 歌单内全量歌曲 ID 有序列表
    songCache: {},        // { id: Song详情 }，按需填充
    ROW_HEIGHT: 56,       // 列表项固定行高（与 CSS .song-list-item height 一致）
    WINDOW_SIZE: 50,      // 虚拟窗口同时渲染的条数
    _scrollTimer: null,   // 滚动节流定时器
    currentIndex: 0,
    mode: 'order', // order | random | single-loop
    randomOrder: [],
    randomPlayed: 0,
    saveTimer: null,
    isSeeking: false,
    hasUserInteracted: false,
    consecutiveErrors: 0,
    loading: false,
    loadSeq: 0,

    init: function(options) {
      this.options = options;
      this.mode = 'order';
      this.bindVolume();
      this.bindEvents();
      this.bindKeyboard();
      this.loadData();
    },

    loadData: function() {
      var self = this;
      var identityId = this.options.identityId;
      var playlistId = this.options.playlistId;

      if (!identityId || !playlistId) {
        // 缺少参数，进入选择流程
        this.openIdentityModal();
        return;
      }

      if (this.loading) {
        return;
      }
      this.loading = true;
      this.loadSeq += 1;
      var seq = this.loadSeq;

      // 清理旧的自动保存定时器，避免切换歌单后叠加
      if (this.saveTimer) {
        clearInterval(this.saveTimer);
        this.saveTimer = null;
      }

      this.hideLoadError();
      $(this.options.titleEl).text('加载中...');
      $(this.options.artistEl).text('-');

      var playlistData = null;
      var progressData = null;
      var stateData = null;
      var doneCount = 0;
      var failed = false;

      function onSettled() {
        doneCount += 1;
        if (doneCount < 3) {
          return;
        }
        if (seq !== self.loadSeq) {
          return; // 已被更新的加载取代
        }
        self.loading = false;
        if (!failed) {
          self.initPlayer(playlistData, progressData, stateData);
        }
      }

      function onFail(message) {
        if (seq !== self.loadSeq || failed) {
          return;
        }
        failed = true;
        self.loading = false;
        self.showLoadError(message);
      }

      MultiTune.get('/playlists/' + encodeURIComponent(playlistId), function(err, data) {
        if (seq !== self.loadSeq) {
          return;
        }
        if (err) {
          onFail('歌单加载失败：' + err);
        } else {
          playlistData = data;
        }
        onSettled();
      });

      // 歌单记忆点：上次播放到哪首歌、播到第几秒
      MultiTune.get('/playlists/' + encodeURIComponent(playlistId) + '/progress', function(err, data) {
        if (seq !== self.loadSeq) {
          return;
        }
        if (err) {
          onFail('记忆点加载失败：' + err);
        } else {
          progressData = data;
        }
        onSettled();
      });

      // 身份记忆点仅用于恢复播放模式，失败时静默降级为默认顺序播放
      MultiTune.get('/playback/' + encodeURIComponent(identityId), function(err, data) {
        if (seq !== self.loadSeq) {
          return;
        }
        if (!err && data) {
          stateData = data;
        }
        onSettled();
      });
    },

    showLoadError: function(message) {
      if (this.options.loadErrorText) {
        $(this.options.loadErrorText).text(message || '加载失败');
      }
      if (this.options.loadErrorBar) {
        $(this.options.loadErrorBar).show();
      }
      $(this.options.titleEl).text('加载失败');
    },

    hideLoadError: function() {
      if (this.options.loadErrorBar) {
        $(this.options.loadErrorBar).hide();
      }
    },

    initPlayer: function(playlist, progress, state) {
      var self = this;
      this.playlist = playlist || {};
      // 优先用全量 id 有序列表（新接口）；兜底从旧 songs 数组取 id
      this.songIds = (playlist && playlist.song_ids && playlist.song_ids.length) ? playlist.song_ids.slice(0) : [];
      this.songCache = {};
      if (playlist && playlist.songs && playlist.songs.length) {
        // 兼容：旧接口返回的 songs 详情直接填进 cache
        for (var k = 0; k < playlist.songs.length; k++) {
          this.songCache[playlist.songs[k].id] = playlist.songs[k];
        }
        // 若没有 song_ids，用 songs 的 id 顺序兜底
        if (this.songIds.length === 0) {
          for (var m = 0; m < playlist.songs.length; m++) {
            this.songIds.push(playlist.songs[m].id);
          }
        }
      }

      $(this.options.playlistNameEl).text(this.playlist.name || '未命名歌单');

      if (this.songIds.length === 0) {
        $(this.options.titleEl).text('暂无歌曲');
        $(this.options.artistEl).text('请先在完整版或 PC 端添加歌曲');
        return;
      }

      // 播放模式来自身份记忆点
      if (state && state.mode && (state.mode === 'order' || state.mode === 'random' || state.mode === 'single-loop')) {
        this.mode = state.mode;
      }

      // 起始点来自歌单记忆点：命中则续播，否则从第一首开始
      var startIndex = 0;
      var startPosition = 0;
      if (progress && progress.song_id) {
        for (var i = 0; i < this.songIds.length; i++) {
          if (this.songIds[i] === progress.song_id) {
            startIndex = i;
            startPosition = progress.position || 0;
            break;
          }
        }
      }

      this.updateModeBtn();
      this.renderSongList();
      this.consecutiveErrors = 0;
      this.playSong(startIndex, false, startPosition);

      // 自动保存播放状态（每 10 秒）
      this.saveTimer = setInterval(function() {
        self.saveState(false);
      }, 10000);

      // 页面离开前保存
      $(window).off('beforeunload.multitune').on('beforeunload.multitune', function() {
        self.saveState(true);
      });
    },

    bindEvents: function() {
      var self = this;
      var audio = $(this.options.audioEl)[0];

      $(this.options.playBtn).on('click', function() {
        self.hasUserInteracted = true;
        self.togglePlay();
      });

      $(this.options.prevBtn).on('click', function() {
        self.hasUserInteracted = true;
        self.playPrev();
      });

      $(this.options.nextBtn).on('click', function() {
        self.hasUserInteracted = true;
        self.playNext();
      });

      $(this.options.modeBtn).on('click', function() {
        self.hasUserInteracted = true;
        self.toggleMode();
      });

      $(this.options.progressBar).on('input', function() {
        self.isSeeking = true;
      });

      $(this.options.progressBar).on('change', function() {
        self.isSeeking = false;
        self.seek(this.value);
      });

      $(this.options.toggleListBtn).on('click', function() {
        self.openSongList();
      });

      if (this.options.switchIdentityBtn) {
        $(this.options.switchIdentityBtn).on('click', function() {
          if (self.loading) {
            return;
          }
          self.openIdentityModal();
        });
      }

      if (this.options.switchPlaylistBtn) {
        $(this.options.switchPlaylistBtn).on('click', function() {
          if (self.loading) {
            return;
          }
          if (self.options.identityId) {
            self.openPlaylistModal(self.options.identityId);
          } else {
            self.openIdentityModal();
          }
        });
      }

      if (this.options.retryLoadBtn) {
        $(this.options.retryLoadBtn).on('click', function() {
          self.hasUserInteracted = true;
          self.loadData();
        });
      }

      $(audio).on('timeupdate', function() {
        self.updateProgress();
      });

      $(audio).on('loadedmetadata', function() {
        self.consecutiveErrors = 0;
        self.updateProgress();
      });

      $(audio).on('ended', function() {
        self.onSongEnded();
      });

      $(audio).on('error', function() {
        self.onAudioError();
      });

      $(audio).on('play', function() {
        self.updatePlayBtn(true);
      });

      $(audio).on('pause', function() {
        self.updatePlayBtn(false);
      });
    },

    bindKeyboard: function() {
      var self = this;
      $(document).on('keydown', function(e) {
        var keyCode = e.which || e.keyCode;
        var targetTag = (e.target && e.target.tagName) ? e.target.tagName.toLowerCase() : '';

        // 输入框内不拦截，避免影响输入
        if (targetTag === 'input' || targetTag === 'textarea') {
          return;
        }

        if (keyCode === 32) {
          // 空格：播放/暂停
          e.preventDefault();
          self.hasUserInteracted = true;
          self.togglePlay();
        } else if (keyCode === 37) {
          // 左方向键：上一曲
          e.preventDefault();
          self.hasUserInteracted = true;
          self.playPrev();
        } else if (keyCode === 39) {
          // 右方向键：下一曲
          e.preventDefault();
          self.hasUserInteracted = true;
          self.playNext();
        }
      });
    },

    bindVolume: function() {
      var self = this;
      var audio = $(this.options.audioEl)[0];
      $(this.options.volumeBar).on('input change', function() {
        var val = parseInt(this.value, 10) || 0;
        audio.volume = val / 100;
      });
    },

    // 取某下标的歌曲详情（可能为 undefined，调用方需判断）
    getSong: function(index) {
      if (index < 0 || index >= this.songIds.length) {
        return null;
      }
      return this.songCache[this.songIds[index]] || null;
    },

    // 按需查询缺失详情并缓存，完成后回调。
    // 去重：同一批次进行中的请求不重复发。
    // _pending: { id: true } 正在查的 id 集合
    ensureSongs: function(ids, callback) {
      var self = this;
      var missing = [];
      if (!this._pending) { this._pending = {}; }
      for (var i = 0; i < ids.length; i++) {
        var id = ids[i];
        if (id && !this.songCache[id] && !this._pending[id]) {
          missing.push(id);
          this._pending[id] = true;
        }
      }
      if (missing.length === 0) {
        if (callback) { callback(); }
        return;
      }
      MultiTune.post('/songs/batch', { ids: missing }, function(err, data) {
        // 无论成败，先清 pending 标记
        for (var j = 0; j < missing.length; j++) {
          delete self._pending[missing[j]];
        }
        if (err || !data || !data.songs) {
          if (callback) { callback(err || '批量查询失败'); }
          return;
        }
        for (var k = 0; k < data.songs.length; k++) {
          self.songCache[data.songs[k].id] = data.songs[k];
        }
        if (callback) { callback(); }
      });
    },

    playSong: function(index, autoPlay, startPosition) {
      if (index < 0 || index >= this.songIds.length) {
        return;
      }

      this.currentIndex = index;
      var self = this;
      var songId = this.songIds[index];

      // 预加载：当前首前后各 3 首的详情，减少切歌等待
      var preloadIds = [];
      for (var p = index - 3; p <= index + 3; p++) {
        if (p >= 0 && p < this.songIds.length) {
          preloadIds.push(this.songIds[p]);
        }
      }
      this.ensureSongs(preloadIds, function() {
        self._doPlaySong(index, autoPlay, startPosition);
      });
    },

    // 实际执行播放（详情已就绪）。从 playSong 拆出，供 ensureSongs 回调调用。
    _doPlaySong: function(index, autoPlay, startPosition) {
      var song = this.getSong(index);
      if (!song) {
        // 详情查不到（歌曲可能已被删除），跳下一首
        this.onAudioError();
        return;
      }
      var audio = $(this.options.audioEl)[0];

      $(this.options.titleEl).text(song.title || '未知歌曲');
      $(this.options.artistEl).text(song.artist || '-');
      $(this.options.coverEl).html('<i class="fas fa-music"></i>');

      audio.src = '/api/songs/' + encodeURIComponent(song.id) + '/stream';
      audio.load();

      var self = this;
      if (startPosition && startPosition > 0) {
        $(audio).one('loadedmetadata', function() {
          try {
            audio.currentTime = startPosition;
          } catch (e) {
            // 部分格式不支持精确 seek，忽略
          }
        });
      }

      this.renderSongList();

      if (autoPlay || this.hasUserInteracted) {
        // 用户交互后才能自动播放，否则等待用户点击
        var playPromise = audio.play();
        if (playPromise && typeof playPromise.then === 'function') {
          playPromise.catch(function() {
            self.updatePlayBtn(false);
          });
        }
      } else {
        this.updatePlayBtn(false);
      }

      this.saveState(false);
    },

    togglePlay: function() {
      var audio = $(this.options.audioEl)[0];
      if (audio.paused) {
        var self = this;
        var playPromise = audio.play();
        if (playPromise && typeof playPromise.then === 'function') {
          playPromise.catch(function() {
            self.updatePlayBtn(false);
          });
        }
      } else {
        audio.pause();
      }
    },

    updatePlayBtn: function(isPlaying) {
      var iconClass = isPlaying ? 'fa-pause' : 'fa-play';
      $(this.options.playBtn).html('<i class="fas ' + iconClass + '"></i>');
    },

    playNext: function() {
      if (this.mode === 'single-loop') {
        this.playSong(this.currentIndex, true, 0);
        return;
      }

      var nextIndex = this.getNextIndex();
      if (nextIndex === -1) {
        // 顺序播放到末尾，回到第一首并暂停
        var audio = $(this.options.audioEl)[0];
        audio.pause();
        this.currentIndex = 0;
        this.playSong(0, false, 0);
        return;
      }

      this.playSong(nextIndex, true, 0);
    },

    playPrev: function() {
      if (this.mode === 'single-loop') {
        this.playSong(this.currentIndex, true, 0);
        return;
      }

      if (this.mode === 'random') {
        // 随机模式上一首：回到上一首随机的歌曲较复杂，简化为随机一首
        var nextIndex = this.getRandomIndex();
        this.playSong(nextIndex, true, 0);
        return;
      }

      var prevIndex = this.currentIndex - 1;
      if (prevIndex < 0) {
        prevIndex = this.songIds.length - 1;
      }
      this.playSong(prevIndex, true, 0);
    },

    onSongEnded: function() {
      if (this.mode === 'single-loop') {
        this.playSong(this.currentIndex, true, 0);
      } else {
        this.playNext();
      }
    },

    getNextIndex: function() {
      if (this.mode === 'random') {
        return this.getRandomIndex();
      }

      var next = this.currentIndex + 1;
      if (next >= this.songIds.length) {
        return -1;
      }
      return next;
    },

    getRandomIndex: function() {
      if (this.songIds.length <= 1) {
        return 0;
      }

      if (this.randomOrder.length === 0 || this.randomPlayed >= this.randomOrder.length) {
        this.buildRandomOrder();
        this.randomPlayed = 0;
      }

      var idx = this.randomOrder[this.randomPlayed];
      this.randomPlayed += 1;
      return idx;
    },

    buildRandomOrder: function() {
      var arr = [];
      for (var i = 0; i < this.songIds.length; i++) {
        arr.push(i);
      }
      // Fisher-Yates shuffle
      for (var j = arr.length - 1; j > 0; j--) {
        var k = Math.floor(Math.random() * (j + 1));
        var tmp = arr[j];
        arr[j] = arr[k];
        arr[k] = tmp;
      }
      // 避免第一首与当前重复
      if (arr[0] === this.currentIndex && arr.length > 1) {
        arr[0] = arr[arr.length - 1];
        arr[arr.length - 1] = this.currentIndex;
      }
      this.randomOrder = arr;
    },

    toggleMode: function() {
      if (this.mode === 'order') {
        this.mode = 'random';
        this.buildRandomOrder();
        this.randomPlayed = 0;
      } else if (this.mode === 'random') {
        this.mode = 'single-loop';
      } else {
        this.mode = 'order';
      }
      this.updateModeBtn();
      this.saveState(true);
    },

    updateModeBtn: function() {
      var iconClass = 'fa-arrow-right';
      if (this.mode === 'random') {
        iconClass = 'fa-random';
      } else if (this.mode === 'single-loop') {
        iconClass = 'fa-redo-alt';
      }
      $(this.options.modeBtn).find('#modeIcon').attr('class', 'fas ' + iconClass);
    },

    seek: function(value) {
      var audio = $(this.options.audioEl)[0];
      if (!audio.duration || isNaN(audio.duration)) {
        return;
      }
      var time = (parseFloat(value) / 100) * audio.duration;
      try {
        audio.currentTime = time;
      } catch (e) {
        // ignore
      }
      this.updateProgress();
    },

    updateProgress: function() {
      var audio = $(this.options.audioEl)[0];
      if (!audio.duration || isNaN(audio.duration)) {
        return;
      }

      var current = audio.currentTime || 0;
      var total = audio.duration;

      if (!this.isSeeking) {
        var percent = total > 0 ? (current / total) * 100 : 0;
        $(this.options.progressBar).val(percent);
      }

      $(this.options.currentTimeEl).text(this.formatTime(current));
      $(this.options.totalTimeEl).text(this.formatTime(total));
    },

    formatTime: function(seconds) {
      if (!seconds || isNaN(seconds)) {
        return '0:00';
      }
      var s = Math.floor(seconds);
      var m = Math.floor(s / 60);
      s = s % 60;
      return m + ':' + (s < 10 ? '0' + s : s);
    },

    onAudioError: function() {
      var self = this;
      this.consecutiveErrors += 1;

      if (this.consecutiveErrors >= 5) {
        $(this.options.titleEl).text('连续多首歌曲加载失败');
        $(this.options.artistEl).text('请检查存储设备是否正常连接');
        var audio = $(this.options.audioEl)[0];
        audio.pause();
        this.updatePlayBtn(false);
        this.consecutiveErrors = 0;
        return;
      }

      $(this.options.titleEl).text('歌曲加载失败');
      $(this.options.artistEl).text('3 秒后自动切换下一首');
      setTimeout(function() {
        self.playNext();
      }, 3000);
    },

    openSongList: function() {
      if (this.options.songListModal) {
        var self = this;
        var $modal = $(this.options.songListModal);
        $modal.one('shown.bs.modal', function() {
          self.scrollActiveSongIntoView();
        });
        $modal.modal('show');
      }
    },

    closeSongList: function() {
      if (this.options.songListModal) {
        $(this.options.songListModal).modal('hide');
      }
    },

    scrollActiveSongIntoView: function() {
      var $list = $(this.options.songListEl);
      if (this.songIds.length === 0) {
        return;
      }
      var listHeight = $list.height();
      // 按 index × 行高直接算目标 scrollTop，不依赖 DOM 测量（虚拟列表下不可靠）
      var targetTop = this.currentIndex * this.ROW_HEIGHT - (listHeight - this.ROW_HEIGHT) / 2;
      if (targetTop < 0) {
        targetTop = 0;
      }
      var self = this;
      $list.stop(true, true).animate({ scrollTop: targetTop }, 200);
      // 动画结束后刷新窗口，确保 active 项可见
      setTimeout(function() { self._renderWindow(); }, 220);
    },

    openModal: function(modalSelector) {
      if (modalSelector) {
        $(modalSelector).modal('show');
      }
    },

    closeModal: function(modalSelector) {
      if (modalSelector) {
        $(modalSelector).modal('hide');
      }
    },

    openIdentityModal: function() {
      var self = this;
      this.openModal(this.options.identityModal);
      var $list = $(this.options.identityListEl);
      $list.html('<div class="loading">正在加载身份...</div>');

      MultiTune.get('/identities?limit=100', function(err, data) {
        if (err) {
          MultiTune.showError($list, '加载身份失败：' + err);
          return;
        }
        self.renderIdentityList(data && data.items ? data.items : []);
      });
    },

    closeIdentityModal: function() {
      this.closeModal(this.options.identityModal);
    },

    renderIdentityList: function(items) {
      var self = this;
      var $list = $(this.options.identityListEl);
      if (!items || items.length === 0) {
        MultiTune.showEmpty($list, '暂无身份，请在完整版或 PC 端创建');
        return;
      }

      var html = '';
      for (var i = 0; i < items.length; i++) {
        var id = items[i];
        var color = id.avatar_color || '#6366f1';
        var name = id.name || '未命名';
        html += '<div class="identity-select-item" data-id="' + escapeHtml(id.id) + '" style="background:' + color + '">';
        html += '<div class="identity-select-inner">';
        html += '<div class="identity-select-name">' + escapeHtml(name) + '</div>';
        html += '</div>';
        html += '</div>';
      }
      $list.html(html);

      $list.find('.identity-select-item').on('click', function() {
        var selectedId = $(this).attr('data-id');
        self.closeIdentityModal();
        self.openPlaylistModal(selectedId);
      });
    },

    openPlaylistModal: function(identityId) {
      var self = this;
      this.options.identityId = identityId;
      this.openModal(this.options.playlistModal);
      var $list = $(this.options.playlistListEl);
      $list.html('<div class="loading">正在加载歌单...</div>');

      var playlistsData = null;
      var stateData = null;
      var stateFailed = false;
      var failed = false;
      var settled = 0;

      function tryRender() {
        if (failed || settled < 2) {
          return;
        }
        self.renderPlaylistList(
          playlistsData && playlistsData.items ? playlistsData.items : [],
          stateData,
          stateFailed,
          identityId
        );
      }

      MultiTune.get('/identities/' + encodeURIComponent(identityId) + '/playlists?limit=100', function(err, data) {
        settled += 1;
        if (err) {
          failed = true;
          $list.html('<div class="error">加载歌单失败：' + escapeHtml(err) + '</div>');
          var $retry = $('<button type="button" class="retry-btn retry-btn-block">重试</button>');
          $retry.on('click', function() {
            self.openPlaylistModal(identityId);
          });
          $list.append($retry);
          return;
        }
        playlistsData = data;
        tryRender();
      });

      // 身份记忆点：用于在歌单列表中标注"上次播放"，失败不阻塞列表
      MultiTune.get('/playback/' + encodeURIComponent(identityId), function(err, data) {
        settled += 1;
        if (err) {
          stateFailed = true;
        } else {
          stateData = data;
        }
        tryRender();
      });
    },

    closePlaylistModal: function() {
      this.closeModal(this.options.playlistModal);
    },

    renderPlaylistList: function(items, state, stateFailed, identityId) {
      var self = this;
      var $list = $(this.options.playlistListEl);
      if (!items || items.length === 0) {
        MultiTune.showEmpty($list, '该身份下暂无歌单');
        return;
      }

      var lastPlaylistId = (state && state.playlist_id) ? state.playlist_id : '';

      var html = '';
      if (stateFailed) {
        html += '<div class="modal-warn-bar">记忆点加载失败，无法标注上次播放 <a class="retry-link" id="retryPlaylistState">重试</a></div>';
      }
      for (var i = 0; i < items.length; i++) {
        var pl = items[i];
        var countText = (pl.song_count || 0) + ' 首歌曲';
        var isLast = lastPlaylistId && pl.id === lastPlaylistId;
        html += '<div class="playlist-select-item' + (isLast ? ' last-played' : '') + '" data-id="' + escapeHtml(pl.id) + '">';
        html += '<div class="playlist-select-name">' + escapeHtml(pl.name || '未命名歌单');
        if (isLast) {
          html += '<span class="last-played-badge">上次播放</span>';
        }
        html += '</div>';
        html += '<div class="playlist-select-meta">' + countText + '</div>';
        html += '</div>';
      }
      $list.html(html);

      $list.find('#retryPlaylistState').on('click', function() {
        self.openPlaylistModal(identityId);
      });

      $list.find('.playlist-select-item').on('click', function() {
        if (self.loading) {
          return;
        }
        var playlistId = $(this).attr('data-id');
        $(this).addClass('disabled');
        self.options.playlistId = playlistId;
        self.hasUserInteracted = true;
        self.closePlaylistModal();
        self.updateUrl();
        self.loadData();
      });
    },

    updateUrl: function() {
      var identityId = this.options.identityId;
      var playlistId = this.options.playlistId;
      if (!identityId || !playlistId) {
        return;
      }
      var url = './player.html?identity_id=' + encodeURIComponent(identityId) + '&playlist_id=' + encodeURIComponent(playlistId);
      if (window.history && window.history.replaceState) {
        window.history.replaceState(null, '', url);
      }
    },

    // 初始化虚拟列表骨架（仅调一次）：撑高 spacer + 窗口容器 + 绑定滚动
    renderSongList: function() {
      var self = this;
      var $list = $(this.options.songListEl);

      if (this.songIds.length === 0) {
        MultiTune.showEmpty($list, '暂无歌曲');
        return;
      }

      // 搭建虚拟列表骨架：spacer 撑出总高度（产生滚动条），窗口容器承载可见项
      var totalHeight = this.songIds.length * this.ROW_HEIGHT;
      var skeleton =
        '<div class="song-list-spacer" style="height:' + totalHeight + 'px">' +
          '<div class="song-list-window" id="songListWindow"></div>' +
        '</div>';
      $list.html(skeleton);

      // 滚动节流：scroll 高频触发，老 WebView 上必须节流
      $list.off('scroll.virtuallist').on('scroll.virtuallist', function() {
        if (self._scrollTimer) { clearTimeout(self._scrollTimer); }
        self._scrollTimer = setTimeout(function() {
          self._renderWindow();
        }, 50);
      });

      // 首次渲染窗口（定位到当前播放首附近）
      this._renderWindow();
    },

    // 渲染可视窗口内的条目。根据 scrollTop 计算窗口起止下标，
    // 按需 ensureSongs 缺失详情后，更新 50 个 DOM 节点的内容与位置。
    _renderWindow: function() {
      var self = this;
      var $list = $(this.options.songListEl);
      if (this.songIds.length === 0) { return; }

      var scrollTop = $list.scrollTop();
      var total = this.songIds.length;
      var rowHeight = this.ROW_HEIGHT;
      var windowSize = this.WINDOW_SIZE;

      // 可视区大约起始 index（向上留一个 buffer，避免滚到边界露白）
      var buffer = 10;
      var startIndex = Math.max(0, Math.floor(scrollTop / rowHeight) - buffer);
      var endIndex = Math.min(total, startIndex + windowSize);

      // 当前窗口需要的 id 集合
      var needIds = [];
      for (var i = startIndex; i < endIndex; i++) {
        needIds.push(this.songIds[i]);
      }

      this.ensureSongs(needIds, function() {
        self._paintWindow(startIndex, endIndex);
      });
    },

    // 把 [startIndex, endIndex) 的条目画到窗口容器
    _paintWindow: function(startIndex, endIndex) {
      var $window = $('#songListWindow');
      if ($window.length === 0) { return; }

      // 窗口容器定位到 startIndex 对应的偏移
      $window.css({
        '-webkit-transform': 'translateY(' + (startIndex * this.ROW_HEIGHT) + 'px)',
        'transform': 'translateY(' + (startIndex * this.ROW_HEIGHT) + 'px)'
      });

      var html = '';
      for (var i = startIndex; i < endIndex; i++) {
        var song = this.songCache[this.songIds[i]];
        var title = song ? (song.title || '未知歌曲') : '加载中...';
        var activeClass = i === this.currentIndex ? ' active' : '';
        html += '<div class="song-list-item' + activeClass + '" data-index="' + i + '">';
        html += '<span class="song-list-title">' + escapeHtml(title) + '</span>';
        html += '</div>';
      }
      $window.html(html);

      // 绑定点击（事件委托到窗口容器，避免每次重建绑定）
      var self = this;
      $window.off('click.virtuallist').on('click.virtuallist', '.song-list-item', function() {
        var idx = parseInt($(this).attr('data-index'), 10);
        self.hasUserInteracted = true;
        self.playSong(idx, true, 0);
      });

      // 窗口位置缓存（供调试/判断是否需要重画）
      this._lastStart = startIndex;
      this._lastEnd = endIndex;
    },

    // includeMode 为 true 时附带播放模式（模式切换、页面离开等关键节点）；
    // 周期上报只发 3 个字段以压缩体积
    saveState: function(includeMode) {
      var audio = $(this.options.audioEl)[0];
      if (!this.songIds.length || this.currentIndex >= this.songIds.length) {
        return;
      }

      var position = Math.floor(audio.currentTime || 0);
      var songId = this.songIds[this.currentIndex];

      var data = {
        playlist_id: this.options.playlistId,
        song_id: songId,
        position: position
      };
      if (includeMode) {
        data.mode = this.mode;
      }

      // 静默保存，不处理失败
      MultiTune.post('/playback/' + encodeURIComponent(this.options.identityId), data, function() {
        // ignore
      });
    }
  };

  function escapeHtml(text) {
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(text));
    return div.innerHTML;
  }

})(jQuery);
