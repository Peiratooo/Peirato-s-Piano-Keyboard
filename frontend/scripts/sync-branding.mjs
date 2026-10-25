import {readFile, writeFile, mkdir} from 'node:fs/promises'
import {dirname, resolve} from 'node:path'
import {fileURLToPath} from 'node:url'

// appicon.png at the repository root is the only editable application logo.
// Platform files are generated copies, never independent brand artwork.
const root = fileURLToPath(new URL('../../', import.meta.url))
const logo = await readFile(resolve(root, 'appicon.png'))
async function writeIfChanged(relative, contents) {
    const destination = resolve(root, relative)
    try { if ((await readFile(destination)).equals(contents)) return false } catch (error) {
        if (error.code !== 'ENOENT') throw error
    }
    await mkdir(dirname(destination), {recursive: true})
    await writeFile(destination, contents)
    return true
}
await writeIfChanged('frontend/public/appicon.png', logo)
await writeIfChanged('frontend/public/app-icon.png', logo)
await writeIfChanged('build/appicon.png', logo)
await writeIfChanged('build/appicon.icon/Assets/appicon.png', logo)
// Native desktop icons need only Node's built-ins. The frontend build syncs
// mobile sizes after dependency installation, avoiding concurrent npm ci runs.
if (!process.argv.includes('--native')) {
    const {default: sharp} = await import('sharp')
    await writeIfChanged('build/ios/icon.png', await sharp(logo).removeAlpha().png().toBuffer())
    for (const [density, size] of Object.entries({mdpi: 48, hdpi: 72, xhdpi: 96, xxhdpi: 144, xxxhdpi: 192})) {
        const icon = await sharp(logo).resize(size, size, {fit: 'contain'}).png().toBuffer()
        for (const name of ['ic_launcher.png', 'ic_launcher_round.png']) {
            await writeIfChanged(`build/android/app/src/main/res/mipmap-${density}/${name}`, icon)
        }
    }
}
