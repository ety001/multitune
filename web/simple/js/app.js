(function($) {
  'use strict';

  window.MultiTune = window.MultiTune || {};

  // API 基础路径
  MultiTune.apiBase = '/api';

  // 通用 GET 请求
  MultiTune.get = function(path, callback) {
    $.ajax({
      url: MultiTune.apiBase + path,
      type: 'GET',
      dataType: 'json',
      success: function(resp) {
        if (resp && resp.code === 0) {
          callback(null, resp.data);
        } else {
          callback(resp ? resp.message : '请求失败', null);
        }
      },
      error: function(xhr, status, err) {
        callback(err || status || '网络错误', null);
      }
    });
  };

  // 通用 POST 请求
  MultiTune.post = function(path, data, callback) {
    $.ajax({
      url: MultiTune.apiBase + path,
      type: 'POST',
      contentType: 'application/json',
      data: JSON.stringify(data),
      dataType: 'json',
      success: function(resp) {
        if (resp && resp.code === 0) {
          callback(null, resp.data);
        } else {
          callback(resp ? resp.message : '请求失败', null);
        }
      },
      error: function(xhr, status, err) {
        callback(err || status || '网络错误', null);
      }
    });
  };

  // 格式化时间戳为本地时间
  MultiTune.formatTime = function(ts) {
    if (!ts) return '-';
    var d = new Date(ts * 1000);
    function pad(n) {
      return n < 10 ? '0' + n : '' + n;
    }
    return d.getFullYear() + '-' +
      pad(d.getMonth() + 1) + '-' +
      pad(d.getDate()) + ' ' +
      pad(d.getHours()) + ':' +
      pad(d.getMinutes());
  };

  // 从 URL 取查询参数
  MultiTune.getQuery = function(key) {
    var match = window.location.search.match(new RegExp('[?&]' + key + '=([^&]+)'));
    return match ? decodeURIComponent(match[1]) : '';
  };

  // 显示错误信息
  MultiTune.showError = function($container, message) {
    $container.html('<div class="error">' + (message || '加载失败') + '</div>');
  };

  // 显示空状态
  MultiTune.showEmpty = function($container, message) {
    $container.html('<div class="empty">' + (message || '暂无数据') + '</div>');
  };

})(jQuery);
