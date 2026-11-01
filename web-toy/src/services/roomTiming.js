// Browser-independent timing math for the piano room WebSocket protocol.
// Server and clients do not share an audio clock, so we estimate the server
// wall-clock offset with a low-RTT sample and buffer a little jitter.
export const ROOM_JITTER_MS = 85

export function pongSample(sentAt, receivedAt, serverTime) {
  const serverAt = Date.parse(serverTime)
  const rtt = receivedAt - sentAt
  if (!Number.isFinite(sentAt) || sentAt <= 0 || !Number.isFinite(serverAt) || rtt < 0 || rtt > 3000) return null
  return {rtt, offset: serverAt - (sentAt + rtt / 2)}
}

export function bestClockOffset(samples, fallback = 0) {
  if (!samples.length) return fallback
  return samples.reduce((a,b) => a.rtt <= b.rtt ? a : b).offset
}

export function remoteEventDelay(serverTime, clockOffsetMs, receivedAt = Date.now()) {
  const stamp = Date.parse(serverTime)
  if (!Number.isFinite(stamp)) return 50
  const target = stamp - clockOffsetMs + ROOM_JITTER_MS
  return Math.max(0, Math.min(180, target - receivedAt))
}
