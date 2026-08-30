import assert from 'node:assert/strict'
import test from 'node:test'
import {isNewerVersion} from './update.js'

test('isNewerVersion compares semver versions correctly', () => {
  // Major bump
  assert.equal(isNewerVersion('1.0.0', '0.9.9'), true)
  assert.equal(isNewerVersion('0.9.9', '1.0.0'), false)

  // Minor bump
  assert.equal(isNewerVersion('0.2.0', '0.1.0'), true)
  assert.equal(isNewerVersion('0.1.0', '0.2.0'), false)

  // Patch bump
  assert.equal(isNewerVersion('0.1.1', '0.1.0'), true)
  assert.equal(isNewerVersion('0.1.0', '0.1.1'), false)

  // Equal versions
  assert.equal(isNewerVersion('0.1.0', '0.1.0'), false)
  assert.equal(isNewerVersion('1.2.3', '1.2.3'), false)

  // Handles 'v' prefix
  assert.equal(isNewerVersion('v0.2.0', '0.1.0'), true)
  assert.equal(isNewerVersion('0.2.0', 'v0.1.0'), true)
  assert.equal(isNewerVersion('v0.1.0', 'v0.1.0'), false)

  // Two-digit parts
  assert.equal(isNewerVersion('0.10.0', '0.9.0'), true)
  assert.equal(isNewerVersion('0.1.10', '0.1.9'), true)
})
