import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
const walk = dir => readdirSync(dir, { withFileTypes: true }).flatMap(entry => entry.isDirectory() ? walk(join(dir, entry.name)) : [join(dir, entry.name)])
let failures = 0
for (const file of walk('src').filter(file => file.endsWith('.vue'))) {
  const source = readFileSync(file, 'utf8')
  const imports = [...source.matchAll(/import\s*\{([^}]+)\}\s*from\s*['"]naive-ui['"]/g)].map(match => match[1]).join(',')
  for (const tag of new Set([...source.matchAll(/<n-([a-z][a-z0-9-]*)\b/g)].map(match => match[1]))) {
    const component = 'N' + tag.split('-').map(part => part[0].toUpperCase() + part.slice(1)).join('')
    if (!new RegExp('\\b' + component + '\\b').test(imports)) {
      console.error(file + ': missing explicit ' + component + ' import')
      failures++
    }
  }
}
if (failures) process.exitCode = 1
else console.log('Naive UI template imports verified')
