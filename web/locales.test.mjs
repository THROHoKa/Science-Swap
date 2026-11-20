import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'

function keys(value, path = []) {
  assert.equal(typeof value === 'string' || (
    typeof value === 'object' && value !== null && !Array.isArray(value)
  ), true, `Invalid translation at ${path.join('.')}`)
  if (typeof value === 'string') return [JSON.stringify(path)]
  return [
    JSON.stringify(path),
    ...Object.entries(value).flatMap(([key, child]) => keys(child, [...path, key])),
  ]
}

test('German and English contain the same translation keys', () => {
  const versions = ['de', 'en'].map((locale) => {
    const messages = readFileSync(new URL(`./src/locales/${locale}.json`, import.meta.url), 'utf8')
    return keys(JSON.parse(messages)).sort()
  })
  assert.deepEqual(versions[0], versions[1])
})
