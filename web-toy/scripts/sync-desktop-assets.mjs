import {copyFile, mkdir} from 'node:fs/promises'
const target = new URL('../public/', import.meta.url)
await mkdir(target, {recursive: true})
for (const name of ['appicon.png', 'keyboard_config.json', 'ChordNames.json']) {
    await copyFile(new URL('../../frontend/public/' + name, import.meta.url), new URL(name, target))
}
