import { cpSync, existsSync, mkdirSync, rmSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const scriptDir = dirname(fileURLToPath(import.meta.url))
const webDir = resolve(scriptDir, '..')
const sourceDir = resolve(webDir, 'auth', 'dist')
const targetDir = resolve(webDir, '..', 'internal', 'server', 'auth', 'ui', 'dist')

if (!existsSync(sourceDir)) {
  console.error(`auth build output is missing: ${sourceDir}`)
  console.error('Run `npm run build:auth` first.')
  process.exit(1)
}

rmSync(targetDir, { recursive: true, force: true })
mkdirSync(targetDir, { recursive: true })
cpSync(sourceDir, targetDir, { recursive: true })

console.log(`synced auth dist to ${targetDir}`)
