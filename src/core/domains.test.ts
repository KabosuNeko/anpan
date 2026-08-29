import assert from 'node:assert/strict'
import test from 'node:test'
import {identifySite, isLikelyUrl, isPlaylistUrl, parseUrlInput} from './domains.js'

test('identifySite recognizes major video and music platforms', () => {
  assert.equal(identifySite('https://music.youtube.com/watch?v=123').key, 'youtube')
  assert.equal(identifySite('https://soundcloud.com/artist/track').key, 'soundcloud')
  assert.equal(identifySite('https://tiktok.com/@user/video/123').key, 'tiktok')
  assert.equal(identifySite('https://bandcamp.com/album/xyz').key, 'bandcamp')
})

test('parseUrlInput extracts cleanUrl and time ranges', () => {
  const t1 = parseUrlInput('https://youtu.be/dQw4w9WgXcQ 01:20-03:45')
  assert.equal(t1.cleanUrl, 'https://youtu.be/dQw4w9WgXcQ')
  assert.equal(t1.timeRange, '01:20-03:45')
  assert.equal(t1.timeLabel, '01:20 → 03:45')

  const t2 = parseUrlInput('https://music.youtube.com/watch?v=abc 45-90')
  assert.equal(t2.cleanUrl, 'https://music.youtube.com/watch?v=abc')
  assert.equal(t2.timeRange, '45-90')

  const t3 = parseUrlInput('https://youtu.be/dQw4w9WgXcQ')
  assert.equal(t3.cleanUrl, 'https://youtu.be/dQw4w9WgXcQ')
  assert.equal(t3.timeRange, undefined)
})

test('isPlaylistUrl detects playlists across platforms', () => {
  assert.equal(isPlaylistUrl('https://music.youtube.com/playlist?list=PL4fGSI1pDJn5kI81J1fYWK5eZRl1zJ5kM'), true)
  assert.equal(isPlaylistUrl('https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PL123'), true)
  assert.equal(isPlaylistUrl('https://soundcloud.com/artist/sets/my-album'), true)
  assert.equal(isPlaylistUrl('https://www.youtube.com/watch?v=dQw4w9WgXcQ'), false)
})
