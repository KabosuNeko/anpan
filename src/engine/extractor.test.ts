import assert from 'node:assert/strict'
import test from 'node:test'
import {extractPortions, extractPlaylistPortions, type VideoMeta} from './extractor.js'

test('extractPortions lists all available resolutions without 8-tier truncation', () => {
  const meta: VideoMeta = {
    title: 'Test Video',
    duration: 120,
    formats: [
      {format_id: 'audio1', acodec: 'mp4a', abr: 128, filesize: 2_000_000},
      {format_id: 'v4320', vcodec: 'av01', height: 4320, fps: 60, tbr: 25_000},
      {format_id: 'v2160', vcodec: 'vp9', height: 2160, fps: 60, dynamic_range: 'HDR', tbr: 15_000},
      {format_id: 'v1440', vcodec: 'vp9', height: 1440, fps: 60, tbr: 8_000},
      {format_id: 'v1080', vcodec: 'avc1', height: 1080, fps: 60, tbr: 4_000},
      {format_id: 'v720', vcodec: 'avc1', height: 720, fps: 30, tbr: 2_000},
      {format_id: 'v480', vcodec: 'avc1', height: 480, fps: 30, tbr: 1_000},
      {format_id: 'v360', vcodec: 'avc1', height: 360, fps: 30, tbr: 600},
      {format_id: 'v240', vcodec: 'avc1', height: 240, fps: 30, tbr: 350},
      {format_id: 'v144', vcodec: 'avc1', height: 144, fps: 30, tbr: 150},
    ],
  }

  const portions = extractPortions(meta, {videoContainer: 'mkv', audioFormat: 'flac'})

  // 9 video resolutions + 2 audio choices (flac + mp3) = 11 portions
  const videoPortions = portions.filter(p => p.kind === 'video')
  assert.equal(videoPortions.length, 9)

  // Verify tags and container
  assert.match(videoPortions[0]!.label, /^4320p60 · mkv/)
  assert.match(videoPortions[1]!.label, /^2160p60 HDR · mkv/)
  assert.match(videoPortions[2]!.label, /^1440p60 · mkv/)
  assert.match(videoPortions[3]!.label, /^1080p60 · mkv/)
  assert.match(videoPortions[4]!.label, /^720p · mkv/)
  assert.match(videoPortions[8]!.label, /^144p · mkv/)

  // Verify size tags exist
  for (const vp of videoPortions) {
    assert.match(vp.label, /~[\d.]+ (KB|MB|GB)/)
    assert.deepEqual(vp.ytdlpArgs.slice(-2), ['--merge-output-format', 'mkv'])
  }

  // Audio portions
  const audioPortions = portions.filter(p => p.kind === 'audio')
  assert.equal(audioPortions.length, 2)
  assert.match(audioPortions[0]!.label, /^audio only · flac/)
  assert.match(audioPortions[1]!.label, /^audio only · mp3/)
})

test('extractPlaylistPortions honors videoContainer and audioFormat', () => {
  const plPortions = extractPlaylistPortions({videoContainer: 'webm', audioFormat: 'm4a'})
  assert.equal(plPortions.length, 3) // m4a, webm best, fallback mp3
  assert.match(plPortions[0]!.label, /all tracks · m4a/)
  assert.match(plPortions[1]!.label, /all videos · webm/)
})
