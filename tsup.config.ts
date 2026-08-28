import {defineConfig} from 'tsup'

export default defineConfig({
  entry: ['src/entry.tsx'],
  format: 'esm',
  target: 'node18',
  clean: true,
  banner: {js: '#!/usr/bin/env node'},
})
