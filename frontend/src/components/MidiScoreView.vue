<template>
    <section class="score-panel" :style="{width: width + 'px', height: height + 'px'}"
        :aria-label="scoreDocument?.native ? 'MusicXML 原谱五线谱' : 'MIDI 自动生成五线谱'">
        <div v-show="!fallback" ref="engraving" class="engraving" aria-hidden="true">
            <div ref="scoreSurface" class="score-surface"></div>
        </div>
        <MidiScoreCanvas v-if="fallback" :width="width" :height="height"
            :active="active" :timeline="timeline" :player-state="playerState" :hand-mode="handMode" />
        <div v-if="!fallback" class="playhead" :class="{'is-waiting': playerState.waiting}"
            :style="{left: playheadX + 'px'}" aria-hidden="true"></div>
        <span class="score-caption">{{ pending ? '正在排版五线谱…' : fallback ? '简化演奏谱 · 排版不可用' : scoreDocument?.native ? 'MusicXML 原谱' : 'MIDI 自动排版（推定）' }}</span>
    </section>
</template>

<script setup>
import {computed, nextTick, onBeforeUnmount, onMounted, ref, watch} from 'vue'
import MidiScoreCanvas from './MidiScoreCanvas.vue'
import {midiTimelineToMusicXML} from '../services/midiScoreNotation'

const props = defineProps({
    scoreDocument: {type:Object, default:null},
    timeline: {type:Object, default:null},
    playerState: {type:Object, default:() => ({})},
    width: {type:Number, default:0},
    height: {type:Number, default:0},
    active: {type:Boolean, default:true},
    handMode: {type:String, default:'both'},
})
const engraving=ref(null), scoreSurface=ref(null)
const fallback=ref(false), pending=ref(false)
const playheadX=computed(() => Math.min(Math.max(100, props.width*.27), 240))
let layout=null, anchorPoints=[], mounted=false, layoutRevision=0, frameId=0
let playerClock={ms:0,at:0,state:{}}, clockRevision=null

function captureClock() {
    const state=props.playerState || {}, now=performance.now()
    const discontinuity=clockRevision!==state.clockRevision
    clockRevision=state.clockRevision
    const received=Number(state.receivedAt)
    playerClock={state:{...state},ms:Math.max(0,Number(state.currentMs)||0),
        at: !discontinuity && Number.isFinite(received) ? Math.min(received,now) : now}
    if (state.waiting && Number.isFinite(state.currentStep?.ms)) playerClock.ms=state.currentStep.ms
}
function currentMs(now) {
    const state=playerClock.state
    if (state.scrubbing || state.waiting) return playerClock.ms
    if (state.leadInMs>0) {
        const elapsed=state.status==='playing'?Math.min(200,Math.max(0,now-playerClock.at)):0
        return playerClock.ms-Math.max(0,state.leadInMs-elapsed*(Number(state.speed)||1))
    }
    let ms=playerClock.ms
    if (state.status==='playing') ms+=Math.min(200,Math.max(0,now-playerClock.at))*(Number(state.speed)||1)
    if (state.mode==='follow' && Number.isFinite(state.currentStep?.ms)
        && state.currentStep.ms>=playerClock.ms) ms=Math.min(ms,state.currentStep.ms)
    return Math.min(Math.max(0,ms),Number(state.rightMs)||props.timeline?.durationMs||ms)
}
function getXAtTime(ms) {
    if (!anchorPoints.length) return 0
    if (ms<=anchorPoints[0].ms) return anchorPoints[0].x
    let low=0,high=anchorPoints.length-1
    while (low<high) {
        const mid=(low+high+1)>>1
        if (anchorPoints[mid].ms<=ms) low=mid
        else high=mid-1
    }
    const a=anchorPoints[low], b=anchorPoints[low+1]
    if (!b || b.ms<=a.ms) return a.x
    const ratio=Math.max(0,Math.min(1,(ms-a.ms)/(b.ms-a.ms)))
    return a.x+(b.x-a.x)*ratio
}
function drawPosition(now) {
    if (!mounted || !props.active || !scoreSurface.value || !layout || !anchorPoints.length) return
    const x=getXAtTime(currentMs(now))
    scoreSurface.value.style.transform='translate3d(' + Math.round(playheadX.value-x) + 'px, 0, 0)'
}
function animating() {
    const state=playerClock.state
    return state.status==='playing' && !state.waiting && !state.scrubbing
}
function frame(now) {
    frameId=0
    drawPosition(now)
    if (mounted && props.active && animating()) frameId=requestAnimationFrame(frame)
}
function syncClock() {
    captureClock()
    if (frameId) cancelAnimationFrame(frameId)
    frameId=0
    drawPosition(performance.now())
    if (mounted && props.active && animating()) frameId=requestAnimationFrame(frame)
}
function hideOppositeStaff(engine) {
    const instruments=engine.Sheet?.Instruments || []
    for (const instrument of instruments) {
        instrument.Visible=true
        for (const staff of instrument.Staves || []) staff.Visible=true
    }
    if (props.handMode==='both') return
    if (instruments.length===1 && instruments[0].Staves?.length===2) {
        instruments[0].Staves[props.handMode==='left'?0:1].Visible=false
    } else if (instruments.length===2 && instruments.every(p=>p.Staves?.length===1)) {
        instruments[props.handMode==='left'?0:1].Visible=false
    }
}
function measurePositions(engine) {
    const graphic=engine.GraphicSheet?.MeasureList||[]
    const markers=(props.timeline?.measures||[]).filter(m=>Number.isFinite(m.ms)).slice().sort((a,b)=>a.ms-b.ms)
    const points=[]
    const zoom=engine.zoom
    for (let i=0; i<graphic.length; i++) {
        const measure=graphic[i]?.find(m=>m && m.PositionAndShape)
        if (!measure) continue
        const box=measure.PositionAndShape
        const x=box.AbsolutePosition.x*10*zoom
        const end=(box.AbsolutePosition.x+box.Size.width)*10*zoom
        const ms=markers[i]?.ms
        if (Number.isFinite(ms)) {
            points.push({ms,x})
            if (i===graphic.length-1) points.push({ms:props.timeline?.durationMs||ms+1,x:end})
        }
    }
    if (points.length<2) return []
    return points
}
async function rebuild() {
    const revision=++layoutRevision
    if (!mounted || !props.active || !props.height || !props.width) return
    const source=props.scoreDocument?.xml || midiTimelineToMusicXML(props.timeline)
    if (!source) {
        pending.value=false
        fallback.value=true
        layout=null
        return
    }
    pending.value=true
    fallback.value=false
    await nextTick()
    try {
        // Only load this fairly large library when score view is actually shown.
        const module=await import('opensheetmusicdisplay')
        const OpenSheetMusicDisplay=module.OpenSheetMusicDisplay || module.default?.OpenSheetMusicDisplay
        if (!OpenSheetMusicDisplay) throw new Error('OSMD 模块未提供排版引擎')
        if (!mounted || revision!==layoutRevision || !scoreSurface.value) return
        scoreSurface.value.replaceChildren()
        const engine=new OpenSheetMusicDisplay(scoreSurface.value, {
            backend:'svg', autoResize:false, renderSingleHorizontalStaffline:true,
            drawingParameters:'compacttight', drawTitle:false, drawSubtitle:false,
            drawComposer:false, drawCredits:false, drawPartNames:false,
            drawFingerings:true, drawMeasureNumbers:true, measureNumberInterval:2,
            disableCursor:true, followCursor:false,
        })
        engine.zoom=Math.max(.4,Math.min(1.15,props.height/(props.handMode==='both'?245:160)))
        await engine.load(source)
        if (!mounted || revision!==layoutRevision) return
        hideOppositeStaff(engine)
        engine.render()
        if (!mounted || revision!==layoutRevision) return
        const anchors=measurePositions(engine)
        if (anchors.length<2) throw new Error('五线谱无法建立小节时间定位')
        layout=engine
        anchorPoints=anchors
        pending.value=false
        fallback.value=false
        syncClock()
    } catch (error) {
        if (!mounted || revision!==layoutRevision) return
        console.error('五线谱排版失败，回退至音符预览',error)
        layout=null
        anchorPoints=[]
        pending.value=false
        fallback.value=true
    }
}
watch(() => props.playerState, syncClock, {deep:true})
watch(() => [props.scoreDocument, props.timeline?.midiId,props.handMode,Math.floor(props.height/24),props.active],rebuild)
watch(() => props.width,() => drawPosition(performance.now()))
onMounted(() => {mounted=true; captureClock(); void rebuild()})
onBeforeUnmount(() => {
    mounted=false
    layoutRevision++
    if (frameId) cancelAnimationFrame(frameId)
    layout=null
    anchorPoints=[]
})
</script>

<style scoped>
.score-panel { position:relative; overflow:hidden; background:#1d1d1d; color:#aaa; }
.engraving { position:absolute; inset:0; overflow:hidden; pointer-events:none; }
.score-surface { position:absolute; top:0; left:0; width:max-content; will-change:transform; }
.score-surface :deep(svg) { display:block; max-width:none !important; filter:invert(1) hue-rotate(180deg); }
.playhead { position:absolute; z-index:3; top:9px; bottom:8px; width:2px;
    transform:translateX(-1px); background:#e7a17a; box-shadow:0 0 10px #e7a17a70; pointer-events:none; }
.playhead.is-waiting { background:#f9d1a0; box-shadow:0 0 15px #f9d1a0a0; }
.score-caption { position:absolute; z-index:4; right:10px; bottom:6px; padding:2px 5px;
    border-radius:3px; background:#1d1d1dc9; font-size:10px; color:#9c9c9c; pointer-events:none; }
</style>
