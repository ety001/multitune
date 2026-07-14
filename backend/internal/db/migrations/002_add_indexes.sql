-- 歌单索引
CREATE INDEX IF NOT EXISTS idx_playlists_identity_id ON playlists(identity_id);

-- 歌单-歌曲关联索引
CREATE INDEX IF NOT EXISTS idx_playlist_songs_playlist_id ON playlist_songs(playlist_id);
CREATE INDEX IF NOT EXISTS idx_playlist_songs_song_id ON playlist_songs(song_id);

-- 歌曲索引
CREATE INDEX IF NOT EXISTS idx_songs_path ON songs(path);
CREATE INDEX IF NOT EXISTS idx_songs_source ON songs(source);
CREATE INDEX IF NOT EXISTS idx_songs_title ON songs(title);
CREATE INDEX IF NOT EXISTS idx_songs_artist ON songs(artist);

-- 单曲进度记忆索引
CREATE INDEX IF NOT EXISTS idx_song_progress_identity_id ON song_progress(identity_id);

-- 播放状态索引
CREATE INDEX IF NOT EXISTS idx_playback_states_playlist_id ON playback_states(playlist_id);
CREATE INDEX IF NOT EXISTS idx_playback_states_song_id ON playback_states(song_id);

-- 全局唯一默认身份约束
CREATE UNIQUE INDEX IF NOT EXISTS idx_one_default_identity ON identities(is_default) WHERE is_default = 1;
