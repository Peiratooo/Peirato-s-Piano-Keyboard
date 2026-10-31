/** @returns {ToySDK.Toy | null} */
const sdk = () => typeof window !== 'undefined' && window.toy || null

export async function support(name) {
  try { return !!sdk() && await sdk().isSupport(name) } catch { return false }
}

// Calling container APIs outside the Bilibili application must never break play.
export function watchToyContainer(callback) {
  let off = null
  let closed = false
  const instance = sdk()
  if (instance?.onContainerChange) {
    Promise.resolve(instance.isSupport('onContainerChange')).then(ok => {
      if (ok && !closed) off = instance.onContainerChange(callback)
    }).catch(() => {})
  }
  return () => { closed = true; off?.() }
}

export async function immerseToy() {
  if (!await support('setContainerMode')) return false
  try { await sdk().setContainerMode({orientation:'landscape',immersive:true}); return true }
  catch { return false }
}

export async function shareToy() {
  if (await support('share')) {
    await sdk().share({path:'index.html'})
    return 'shared'
  }
  if (await support('getQrCode')) {
    const result = await sdk().getQrCode({path:'index.html'})
    return result
  }
  return null
}

export async function saveToyPreferences(data) {
  const encoded = JSON.stringify(data)
  try { localStorage.setItem('pp-toy-settings', encoded) } catch {}
  if (new TextEncoder().encode(encoded).length > 1024 || !await support('setCloudStorage')) return false
  try { await sdk().setCloudStorage({preferences:encoded}); return true } catch { return false }
}

export async function loadToyPreferences() {
  let local = {}
  try { local = JSON.parse(localStorage.getItem('pp-toy-settings') || '{}') } catch {}
  if (await support('getCloudStorage')) {
    try {
      const data = await sdk().getCloudStorage(['preferences'])
      if (data.preferences) return {...local,...JSON.parse(data.preferences)}
    } catch {}
  }
  return local
}
