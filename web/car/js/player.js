(function($) {
  'use strict';

  window.MultiTunePlayer = {
    options: null,
    playlist: null,
    songs: [],
    currentIndex: 0,
    mode: 'order', // order | random | single-loop
    randomOrder: [],
    randomPlayed: 0,
    saveTimer: null,
    isSeeking: false,
    hasUserInteracted: false,
    consecutiveErrors: 0,

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

      var playlistLoaded = false;
      var stateLoaded = false;
      var playlistData = null;
      var stateData = null;

      function tryInit() {
        if (playlistLoaded && stateLoaded) {
          self.initPlayer(playlistData, stateData);
        }
      }

      MultiTune.get('/playlists/' + encodeURIComponent(playlistId), function(err, data) {
        playlistLoaded = true;
        if (err) {
          $(self.options.playlistNameEl).text('加载失败');
          $(self.options.titleEl).text('歌单加载失败');
          return;
        }
        playlistData = data;
        tryInit();
      });

      MultiTune.get('/playback/' + encodeURIComponent(identityId), function(err, data) {
        stateLoaded = true;
        if (!err && data) {
          stateData = data;
        }
        tryInit();
      });
    },

    initPlayer: function(playlist, state) {
      var self = this;
      this.playlist = playlist || {};
      this.songs = (playlist && playlist.songs) ? playlist.songs : [];

      $(this.options.playlistNameEl).text(this.playlist.name || '未命名歌单');

      if (this.songs.length === 0) {
        $(this.options.titleEl).text('暂无歌曲');
        $(this.options.artistEl).text('请先在完整版或 PC 端添加歌曲');
        return;
      }

      // 确定起始歌曲
      var startIndex = 0;
      var startPosition = 0;
      if (state) {
        if (state.mode && (state.mode === 'order' || state.mode === 'random' || state.mode === 'single-loop')) {
          this.mode = state.mode;
        }
        if (state.song_id) {
          for (var i = 0; i < this.songs.length; i++) {
            if (this.songs[i].id === state.song_id) {
              startIndex = i;
              startPosition = state.position || 0;
              break;
            }
          }
        }
      }

      this.updateModeBtn();
      this.renderSongList();
      this.consecutiveErrors = 0;
      this.playSong(startIndex, false, startPosition);

      // 自动保存播放状态（每 5 秒）
      this.saveTimer = setInterval(function() {
        self.saveState(false);
      }, 5000);

      // 页面离开前保存
      $(window).on('beforeunload', function() {
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

      if (this.options.closeListBtn) {
        $(this.options.closeListBtn).on('click', function() {
          self.closeSongList();
        });
      }

      if (this.options.songListMask) {
        $(this.options.songListMask).on('click', function() {
          self.closeSongList();
        });
      }

      if (this.options.switchIdentityBtn) {
        $(this.options.switchIdentityBtn).on('click', function() {
          self.openIdentityModal();
        });
      }

      if (this.options.backToPlaylistBtn) {
        $(this.options.backToPlaylistBtn).on('click', function() {
          if (self.options.identityId) {
            self.openPlaylistModal(self.options.identityId);
          } else {
            self.openIdentityModal();
          }
        });
      }

      if (this.options.closeIdentityBtn) {
        $(this.options.closeIdentityBtn).on('click', function() {
          self.closeIdentityModal();
        });
      }

      if (this.options.identityMask) {
        $(this.options.identityMask).on('click', function() {
          self.closeIdentityModal();
        });
      }

      if (this.options.closePlaylistBtn) {
        $(this.options.closePlaylistBtn).on('click', function() {
          self.closePlaylistModal();
        });
      }

      if (this.options.playlistMask) {
        $(this.options.playlistMask).on('click', function() {
          self.closePlaylistModal();
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

    playSong: function(index, autoPlay, startPosition) {
      if (index < 0 || index >= this.songs.length) {
        return;
      }

      this.currentIndex = index;
      var song = this.songs[index];
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

      this.saveState(true);
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
        prevIndex = this.songs.length - 1;
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
      if (next >= this.songs.length) {
        return -1;
      }
      return next;
    },

    getRandomIndex: function() {
      if (this.songs.length <= 1) {
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
      for (var i = 0; i < this.songs.length; i++) {
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
      var text = '顺序播放';
      if (this.mode === 'random') {
        iconClass = 'fa-shuffle';
        text = '随机播放';
      } else if (this.mode === 'single-loop') {
        iconClass = 'fa-rotate-right';
        text = '单曲循环';
      }
      $(this.options.modeBtn).html('<i class="fas ' + iconClass + '"></i> <span>' + text + '</span>');
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
        $(this.options.songListModal).show();
      }
    },

    closeSongList: function() {
      if (this.options.songListModal) {
        $(this.options.songListModal).hide();
      }
    },

    openModal: function(modalSelector) {
      if (modalSelector) {
        $(modalSelector).show();
      }
    },

    closeModal: function(modalSelector) {
      if (modalSelector) {
        $(modalSelector).hide();
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
        html += '<div class="identity-select-name">' + escapeHtml(name) + '</div>';
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

      MultiTune.get('/identities/' + encodeURIComponent(identityId) + '/playlists?limit=100', function(err, data) {
        if (err) {
          MultiTune.showError($list, '加载歌单失败：' + err);
          return;
        }
        self.renderPlaylistList(data && data.items ? data.items : []);
      });
    },

    closePlaylistModal: function() {
      this.closeModal(this.options.playlistModal);
    },

    renderPlaylistList: function(items) {
      var self = this;
      var $list = $(this.options.playlistListEl);
      if (!items || items.length === 0) {
        MultiTune.showEmpty($list, '该身份下暂无歌单');
        return;
      }

      var html = '';
      for (var i = 0; i < items.length; i++) {
        var pl = items[i];
        var countText = (pl.song_count || 0) + ' 首歌曲';
        html += '<div class="playlist-select-item" data-id="' + escapeHtml(pl.id) + '">';
        html += '<div class="playlist-select-name">' + escapeHtml(pl.name || '未命名歌单') + '</div>';
        html += '<div class="playlist-select-meta">' + countText + '</div>';
        html += '</div>';
      }
      $list.html(html);

      $list.find('.playlist-select-item').on('click', function() {
        var playlistId = $(this).attr('data-id');
        self.options.playlistId = playlistId;
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

    renderSongList: function() {
      var self = this;
      var $list = $(this.options.songListEl);
      if (this.songs.length === 0) {
        MultiTune.showEmpty($list, '暂无歌曲');
        return;
      }

      var html = '';
      for (var i = 0; i < this.songs.length; i++) {
        var song = this.songs[i];
        var activeClass = i === this.currentIndex ? ' active' : '';
        html += '<div class="song-list-item' + activeClass + '" data-index="' + i + '">';
        html += '<div class="song-list-title">' + escapeHtml(song.title || '未知歌曲') + '</div>';
        html += '</div>';
      }
      $list.html(html);

      $list.find('.song-list-item').on('click', function() {
        var idx = parseInt($(this).attr('data-index'), 10);
        self.hasUserInteracted = true;
        self.playSong(idx, true, 0);
      });
    },

    saveState: function(force) {
      var audio = $(this.options.audioEl)[0];
      if (!this.songs.length || !this.songs[this.currentIndex]) {
        return;
      }

      var position = Math.floor(audio.currentTime || 0);
      var songId = this.songs[this.currentIndex].id;

      var data = {
        playlist_id: this.options.playlistId,
        song_id: songId,
        position: position,
        mode: this.mode
      };

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
