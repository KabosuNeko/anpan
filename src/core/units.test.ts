import assert from 'node:assert/strict'
import path from 'node:path'
import test from 'node:test'
import {formatBytes, formatDuration, formatSpeed, formatEta, truncate, shortenPath, resolveUserPath, wrapText} from './units.js'

test('formatBytes converts sizes to human units', () => {
  assert.equal(formatBytes(0), '')
  assert.equal(formatBytes(-1), '')
  assert.equal(formatBytes(500), '500 B')
  assert.equal(formatBytes(1024), '1 KB')
  assert.equal(formatBytes(1536), '1.5 KB')
  assert.equal(formatBytes(10240), '10 KB')
  assert.equal(formatBytes(1048576), '1 MB')
  assert.equal(formatBytes(1073741824), '1 GB')
})

test('formatDuration converts seconds to time strings', () => {
  assert.equal(formatDuration(0), '')
  assert.equal(formatDuration(-5), '')
  assert.equal(formatDuration(5), '0:05')
  assert.equal(formatDuration(65), '1:05')
  assert.equal(formatDuration(3661), '1:01:01')
})

test('formatSpeed and formatEta delegate correctly', () => {
  assert.equal(formatSpeed(1048576), '1 MB/s')
  assert.equal(formatSpeed(0), '')
  assert.equal(formatEta(90), '1:30')
  assert.equal(formatEta(0), '')
})

test('truncate clips long text with ellipsis and handles CJK width', () => {
  assert.equal(truncate('hello', 10), 'hello')
  assert.equal(truncate('hello world', 8), 'hello w…')
  // CJK characters take 2 terminal columns each
  assert.equal(truncate('澤野弘之', 6), '澤野…')
})

test('shortenPath replaces homedir with ~ and truncates', () => {
  assert.equal(shortenPath('/home/user/Downloads/file.mp4', '/home/user'), '~/Downloads/file.mp4')
  assert.equal(shortenPath('/other/path/file.mp4', '/home/user'), path.normalize('/other/path/file.mp4'))
})

test('resolveUserPath expands tilde across platforms', () => {
  const fakeHome = path.resolve('/mock/home/user')
  assert.equal(resolveUserPath('~/Music', fakeHome), path.resolve(fakeHome, 'Music'))
  assert.equal(resolveUserPath('~', fakeHome), fakeHome)
  assert.equal(resolveUserPath(fakeHome, fakeHome), fakeHome)
})

test('wrapText breaks long text into lines', () => {
  assert.deepEqual(wrapText('a b c d e', 5), ['a b c', 'd e'])
  assert.deepEqual(wrapText('short', 20), ['short'])
  assert.deepEqual(wrapText('  spaced  out  ', 20), ['spaced out'])
})
