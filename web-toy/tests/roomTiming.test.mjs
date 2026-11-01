import test from 'node:test'
import assert from 'node:assert/strict'
import {pongSample, bestClockOffset, remoteEventDelay, ROOM_JITTER_MS} from '../src/services/roomTiming.js'

test('RTT correction gives server offset independent of network latency', () => {
  const sample = pongSample(1000, 1100, new Date(5050).toISOString())
  assert.deepEqual(sample, {rtt:100, offset:4000})
})
test('clock uses low latency samples rather than average of delayed packets', () => {
  const samples = [{rtt:900,offset:3800}, {rtt:30,offset:4010}, {rtt:80,offset:4000}]
  assert.equal(bestClockOffset(samples),4010)
  assert.equal(bestClockOffset([], 320),320)
})
test('a remote key visual and its sound can use the same buffered delay', () => {
  const receivedAt=2000, offset=500
  const eventAt=new Date(2450).toISOString() // server event happened at local 1950
  assert.equal(remoteEventDelay(eventAt,offset,receivedAt),Math.max(0,ROOM_JITTER_MS-50))
})
test('absurd, missing and late timestamps are clamped safely', () => {
  assert.equal(pongSample(0,100,'bad'),null)
  assert.equal(pongSample(100,4100,new Date(500).toISOString()),null)
  assert.equal(remoteEventDelay('bad',0,2000),50)
  assert.equal(remoteEventDelay(new Date(100000).toISOString(),0,1),180)
  assert.equal(remoteEventDelay(new Date(1).toISOString(),0,100000),0)
})
