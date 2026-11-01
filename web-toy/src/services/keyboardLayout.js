export {createKeyboardLayout} from '../../../frontend/src/services/keyboardLayout'
export const pitchName = midi => ['C', 'C♯', 'D', 'D♯', 'E', 'F', 'F♯', 'G', 'G♯', 'A', 'A♯', 'B'][midi % 12] + (Math.floor(midi / 12) - 1)
