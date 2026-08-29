import assert from 'node:assert/strict'
import test from 'node:test'
import {inspectTarget, parseMagnetName} from './router.js'

test('parseMagnetName extracts display name or fallback', () => {
  const m1 = 'magnet:?xt=urn:btih:d540fc48eb12f2833163eed6421d449dd8f1ce1f&dn=Ubuntu+24.04+LTS+Desktop'
  assert.equal(parseMagnetName(m1), 'Ubuntu 24.04 LTS Desktop')

  const m2 = 'magnet:?xt=urn:btih:1234567890abcdef'
  assert.match(parseMagnetName(m2), /Torrent \([0-9a-f]+\)/i)
})

test('inspectTarget classifies magnet and torrent links', async () => {
  const res1 = await inspectTarget('magnet:?xt=urn:btih:1234567890abcdef&dn=ArchLinux')
  assert.equal(res1.type, 'torrent')
  if (res1.type === 'torrent') {
    assert.equal(res1.name, 'ArchLinux')
  }

  const res2 = await inspectTarget('https://releases.ubuntu.com/noble/ubuntu-24.04.torrent')
  assert.equal(res2.type, 'torrent')
  if (res2.type === 'torrent') {
    assert.equal(res2.name, 'ubuntu-24.04')
  }
})

test('inspectTarget classifies known media sites as video', async () => {
  const res = await inspectTarget('https://music.youtube.com/watch?v=123')
  assert.equal(res.type, 'video')
  if (res.type === 'video') {
    assert.equal(res.cleanUrl, 'https://music.youtube.com/watch?v=123')
  }
})

test('inspectTarget classifies direct file extensions', async () => {
  const res = await inspectTarget('https://example.com/downloads/archlinux-2026.08.01-x86_64.iso')
  assert.equal(res.type, 'direct')
  if (res.type === 'direct') {
    assert.equal(res.filename, 'archlinux-2026.08.01-x86_64.iso')
  }
})
