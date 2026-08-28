import assert from 'node:assert/strict'
import test from 'node:test'
import {parseArgs} from './options.js'

test('parses a single url positional argument', () => {
  assert.deepEqual(parseArgs(['https://example.com/video']), {
    help: false,
    version: false,
    initialUrl: 'https://example.com/video',
  })
})

test('parses help and version flags', () => {
  assert.deepEqual(parseArgs(['--help']), {help: true, version: false})
  assert.deepEqual(parseArgs(['-h']), {help: true, version: false})
  assert.deepEqual(parseArgs(['--version']), {help: false, version: true})
  assert.deepEqual(parseArgs(['-v']), {help: false, version: true})
})

test('rejects multiple urls and unknown options', () => {
  assert.match(parseArgs(['--wat']).error ?? '', /unknown option/)
  assert.match(parseArgs(['one', 'two']).error ?? '', /single url/)
})
