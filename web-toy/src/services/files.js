export function chooseFile(accept) {
  return new Promise(resolve => {
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = accept
    input.hidden = true
    input.onchange = () => { const file = input.files[0]; input.remove(); resolve(file || null) }
    input.oncancel = () => { input.remove(); resolve(null) }
    document.body.append(input)
    input.click()
  })
}

export function downloadFile(blob, name) {
  const url = URL.createObjectURL(blob), link = document.createElement('a')
  link.href = url
  link.download = name
  link.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
