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
      this.loadData();
    },

    loadData: function() {
      var self = this;
      var playlistLoaded = false;
      var stateLoaded = false;
      var playlistData = null;
      var stateData = null;

      function tryInit() {
        if (playlistLoaded && stateLoaded) {
          self.initPlayer(playlistData, stateData);
        }
      }

      MultiTune.get('/playlists/' + encodeURIComponent(self.options.playlistId), function(err, data) {
        playlistLoaded = true;
        if (err) {
          $(self.options.playlistNameEl).text('加载失败');
          $(self.options.titleEl).text('歌单加载失败');
          return;
        }
        playlistData = data;
        tryInit();
      });

      MultiTune.get('/playback/' + encodeURIComponent(self.options.identityId), function(err, data) {
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
        $(this.options.artistEl).text('请先在现代版或 PC 端添加歌曲');
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
        var $list = $(self.options.songListEl);
        if ($list.is(':visible')) {
          $list.hide();
          $(self.options.toggleListBtn).text('展开播放列表');
        } else {
          $list.show();
          $(self.options.toggleListBtn).text('收起播放列表');
        }
      });

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
      $(this.options.coverEl).text((song.title || '♪').charAt(0));

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
      $(this.options.playBtn).text(isPlaying ? '⏸' : '▶');
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
      var text = '顺序播放';
      if (this.mode === 'random') {
        text = '随机播放';
      } else if (this.mode === 'single-loop') {
        text = '单曲循环';
      }
      $(this.options.modeBtn).text(text);
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
        html += '<div class="song-list-artist">' + escapeHtml(song.artist || '-') + '</div>';
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
