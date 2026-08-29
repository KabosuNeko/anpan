import assert from 'node:assert/strict'
import test from 'node:test'
import {parseAriaEta, parseAriaProgressLine, parseUnitBytes} from './aria2c.js'

test('parseUnitBytes parses units accurately', () => {
  assert.equal(parseUnitBytes('0B'), 0)
  assert.equal(parseUnitBytes('500B'), 500)
  assert.equal(parseUnitBytes('10KiB'), 10240)
  assert.equal(parseUnitBytes('1.5MiB'), 1572864)
  assert.equal(parseUnitBytes('1.4GiB'), 1503238554)
})

test('parseAriaEta parses duration formats', () => {
  assert.equal(parseAriaEta('2s'), 2)
  assert.equal(parseAriaEta('5m'), 300)
  assert.equal(parseAriaEta('2m30s'), 150)
  assert.equal(parseAriaEta('1h20m'), 4800)
})

test('parseAriaProgressLine parses direct HTTP downloads', () => {
  const line = '[#2089b0 1.2MiB/4.5MiB(26%) CN:16 DL:1.1MiB ETA:2s]'
  const res = parseAriaProgressLine(line)
  assert.ok(res)
  assert.equal(res.downloadedBytes, 1258291)
  assert.equal(res.totalBytes, 4718592)
  assert.equal(res.connections, 16)
  assert.equal(res.speed, 1153434)
  assert.equal(res.eta, 2)
})

test('parseAriaProgressLine parses BitTorrent downloads with seeders and upload speed', () => {
  const line = '[#a77dff 12MiB/1.4GiB(1%) CN:5 SD:2 DL:1.2MiB UL:32KiB ETA:5m]'
  const res = parseAriaProgressLine(line)
  assert.ok(res)
  assert.equal(res.downloadedBytes, 12582912)
  assert.equal(res.totalBytes, 1503238554)
  assert.equal(res.connections, 5)
  assert.equal(res.seeders, 2)
  assert.equal(res.speed, 1258291)
  assert.equal(res.eta, 300)
})

test('parseAriaProgressLine parses BitTorrent initial state with 0% and 0 seeders', () => {
  const line = '[#a77dff 0B/1.4GiB(0%) CN:5 SD:0 DL:0B]'
  const res = parseAriaProgressLine(line)
  assert.ok(res)
  assert.equal(res.downloadedBytes, 0)
  assert.equal(res.totalBytes, 1503238554)
  assert.equal(res.connections, 5)
  assert.equal(res.seeders, 0)
  assert.equal(res.speed, 0)
})

test('parseAriaProgressLine parses Magnet metadata downloading phase', () => {
  const line = '[#13d9df 0B/0B CN:0 SD:0 DL:0B]'
  const res = parseAriaProgressLine(line)
  assert.ok(res)
  assert.equal(res.downloadedBytes, 0)
  assert.equal(res.totalBytes, undefined)
  assert.equal(res.connections, 0)
  assert.equal(res.seeders, 0)
  assert.equal(res.speed, 0)
})
