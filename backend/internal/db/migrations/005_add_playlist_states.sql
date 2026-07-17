-- 歌单播放记忆表（每个歌单一条：上次播放到哪首歌、播到第几秒）
CREATE TABLE IF NOT EXISTS playlist_states (
    playlist_id TEXT PRIMARY KEY,
    song_id TEXT,
    position INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
    FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE SET NULL
);
