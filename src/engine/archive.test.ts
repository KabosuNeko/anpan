import assert from 'node:assert/strict'
import test from 'node:test'
import {isArchivePostUrl, parseArchiveUrl} from './archive.js'

test('isArchivePostUrl detects Kemono, Coomer, and Pawchive post URLs', () => {
  const validUrls = [
    'https://kemono.cr/patreon/user/90822862/post/147648418',
    'https://kemono.su/fanbox/user/3873554/post/12509033',
    'https://kemono.party/subscribestar/user/123/post/456',
    'https://coomer.su/onlyfans/user/model1/post/9999',
    'https://coomer.st/fansly/user/model2/post/8888',
    'https://pawchive.st/fanbox/user/3873554/post/12509033',
    'https://pawchive.pw/patreon/user/555/post/777',
    'http://pawchive.st/fantia/user/999/post/111',
  ]

  for (const url of validUrls) {
    assert.ok(isArchivePostUrl(url), `Should recognize ${url}`)
  }

  const invalidUrls = [
    'https://youtube.com/watch?v=dQw4w9WgXcQ',
    'https://kemono.cr/artists',
    'https://kemono.cr/patreon/user/90822862',
    'https://pawchive.st/',
    'https://pawchive.pw/fanbox/user/3873554',
    'https://example.com/file.zip',
  ]

  for (const url of invalidUrls) {
    assert.ok(!isArchivePostUrl(url), `Should not recognize ${url}`)
  }
})

test('parseArchiveUrl extracts domain, service, user, and id', () => {
  const kemonoParsed = parseArchiveUrl('https://kemono.cr/patreon/user/90822862/post/147648418')
  assert.deepEqual(kemonoParsed, {
    domain: 'kemono.cr',
    service: 'patreon',
    user: '90822862',
    id: '147648418',
  })

  const pawchiveParsed = parseArchiveUrl('https://pawchive.pw/fanbox/user/3873554/post/12509033')
  assert.deepEqual(pawchiveParsed, {
    domain: 'pawchive.pw',
    service: 'fanbox',
    user: '3873554',
    id: '12509033',
  })

  const coomerParsed = parseArchiveUrl('https://coomer.st/onlyfans/user/model_abc/post/123456')
  assert.deepEqual(coomerParsed, {
    domain: 'coomer.st',
    service: 'onlyfans',
    user: 'model_abc',
    id: '123456',
  })
})
