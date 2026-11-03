<template>
    <section class="score-panel" :style="{width: width + 'px', height: height + 'px'}" aria-label="五线谱" data-piano-control @wheel="browseWheel">
        <div ref="viewport" class="engraving" :class="{'is-dragging': dragging}" tabindex="0" aria-label="横向浏览乐谱"
            title="拖动或滚轮横向浏览；Alt + 滚轮纵向浏览" @scroll.passive="onViewportScroll"
            @pointerdown="beginPan" @pointermove="movePan" @pointerup="endPan"
            @pointerleave="!dragging && endPan($event)"
            @pointercancel="endPan" @lostpointercapture="endPan" @keydown="browseByKey"
            @click="seekNote" @keydown.enter.prevent="seekNote" @keydown.space.prevent="seekNote">
            <div ref="scoreSurface" class="score-surface" :style="{marginLeft: musicStart - firstNoteX + 'px', top: sourceTop + 'px'}"></div>
        </div>
        <div v-show="layoutReady" class="signature-panel" :style="{width: signatureWidth + 'px'}" aria-label="谱号、调号与拍号">
            <div ref="signatureSurface" class="signature-surface" :style="{top: sourceTop - scrollY + 'px'}"></div>
        </div>
        <button v-if="browsing && layoutReady" class="resume-follow" @click="resumeFollow">回到播放位置</button>
        <div v-if="layoutReady" v-show="cursorLeft >= musicStart - 14 && cursorLeft <= width - 8" class="playhead" :class="{'is-waiting': playerState.waiting}"
            :style="{left: cursorLeft + 'px'}" aria-hidden="true"></div>
        <p v-if="error" class="score-message" role="status">{{error}}</p>
        <span class="score-caption">{{pending ? '正在排版…' : scoreDocument?.native ? '原谱' : timeline?.keySignatures?.length ? '自动记谱' : '自动记谱 · 文件未提供调号'}}</span>
    </section>
</template>

<script setup>
import {computed, nextTick, onBeforeUnmount, onMounted, ref, watch} from 'vue'
import {Key} from 'tonal'
import {midiTimelineToScore} from '../services/midiScoreNotation'

const props = defineProps({
    scoreDocument: {type: Object, default: null}, timeline: {type: Object, default: null},
    playerState: {type: Object, default: () => ({})}, width: {type: Number, default: 0},
    height: {type: Number, default: 0}, active: {type: Boolean, default: true},
    handMode: {type: String, default: 'both'},
})
const emit = defineEmits(['seek'])
const viewport = ref(null), scoreSurface = ref(null), signatureSurface = ref(null)
const pending = ref(false), error = ref(''), layoutReady = ref(false)
const browsing = ref(false), dragging = ref(false)
const signatureWidth = ref(150), firstNoteX = ref(0), cursorLeft = ref(0), sourceTop = ref(0), scrollY = ref(0)
const musicStart = computed(() => signatureWidth.value + 24)
const colorOptions = {applyToNoteheads: true, applyToStem: true, applyToFlag: true,
    applyToLedgerLines: true, applyToBeams: false, applyToTies: false, applyToSlurs: false}
let anchorPoints = [], mounted = false, layoutRevision = 0, activeColumn = -1, frameId = 0, firstColumn = 0
let clock = {ms: 0, at: 0, state: {}}, presentedMs = 0, forcePosition = true
let pageOffset = 0, motion = null, signatureKey = '', cropTop = 0, svgHeight = 0
let panGesture = null, ignoreNextClick = false, playbackScrollLeft = 0

function seekNote(event) {
    if (event.type === 'click' && ignoreNextClick) {ignoreNextClick = false; return}
    const note = event.target.closest('[data-score-ms]')
    if (!note) return
    viewport.value.scrollTo({left: viewport.value.scrollLeft, top: viewport.value.scrollTop, behavior: 'instant'})
    playbackScrollLeft = viewport.value.scrollLeft
    browsing.value = false
    forcePosition = true
    emit('seek', Number(note.dataset.scoreMs))
}

function beginBrowse() {
    browsing.value = true
    motion = null
    pageOffset = viewport.value.scrollLeft
}

function onViewportScroll() {
    scrollY.value = viewport.value.scrollTop
    if (layoutReady.value && !browsing.value && Math.abs(viewport.value.scrollLeft - playbackScrollLeft) > 1) beginBrowse()
    if (!browsing.value || !anchorPoints.length) return
    pageOffset = viewport.value.scrollLeft
    if (activeColumn >= 0) cursorLeft.value = anchorPoints[activeColumn].x + musicStart.value - firstNoteX.value - pageOffset
    let low = 0, high = anchorPoints.length - 1
    const left = firstNoteX.value + pageOffset
    while (low < high) {
        const middle = (low + high + 1) >> 1
        if (anchorPoints[middle].x <= left) low = middle
        else high = middle - 1
    }
    drawSignature(anchorPoints[low].signature)
}

function browseWheel(event) {
    if (!layoutReady.value || event.ctrlKey || event.metaKey) return
    event.preventDefault()
    event.stopPropagation()
    beginBrowse()
    const vertical = event.altKey
    const unit = event.deltaMode === 1 ? 16 : event.deltaMode === 2
        ? (vertical ? viewport.value.clientHeight : viewport.value.clientWidth) : 1
    const delta = (Math.abs(event.deltaX) > Math.abs(event.deltaY) ? event.deltaX : event.deltaY) * unit
    viewport.value.scrollBy({left: vertical ? 0 : delta, top: vertical ? delta : 0, behavior: 'smooth'})
}

function beginPan(event) {
    if (!layoutReady.value || panGesture) return
    ignoreNextClick = false
    const element = viewport.value, box = element.getBoundingClientRect()
    if (event.clientX - box.left >= element.clientWidth || event.clientY - box.top >= element.clientHeight || event.button === 1) {
        beginBrowse()
        return
    }
    if (event.button !== 0) return
    panGesture = {id: event.pointerId, x: event.clientX, y: event.clientY,
        left: element.scrollLeft, top: element.scrollTop}
}

function movePan(event) {
    if (panGesture?.id !== event.pointerId) return
    const dx = event.clientX - panGesture.x, dy = event.clientY - panGesture.y
    if (!dragging.value && Math.hypot(dx, dy) < 6) return
    if (!dragging.value) {
        beginBrowse()
        dragging.value = true
        ignoreNextClick = true
        viewport.value.setPointerCapture(event.pointerId)
    }
    event.preventDefault()
    viewport.value.scrollTo({left: panGesture.left - dx, top: panGesture.top - dy, behavior: 'instant'})
}

function endPan(event) {
    if (panGesture?.id !== event.pointerId) return
    panGesture = null
    dragging.value = false
    if (viewport.value.hasPointerCapture(event.pointerId)) viewport.value.releasePointerCapture(event.pointerId)
}

function browseByKey(event) {
    if (!layoutReady.value) return
    const width = viewport.value.clientWidth - musicStart.value
    const distance = {ArrowLeft: -60, ArrowRight: 60, PageUp: -width, PageDown: width}
    let left
    if (event.key === 'Home') left = 0
    else if (event.key === 'End') left = anchorPoints.at(-1)?.x - firstNoteX.value
    else if (event.key in distance) left = viewport.value.scrollLeft + distance[event.key]
    else return
    event.preventDefault()
    beginBrowse()
    viewport.value.scrollTo({left, behavior: 'smooth'})
}

function resumeFollow() {
    browsing.value = false
    motion = null
    viewport.value.scrollTo({left: viewport.value.scrollLeft, top: viewport.value.scrollTop, behavior: 'instant'})
    playbackScrollLeft = viewport.value.scrollLeft
    forcePosition = true
    if (!frameId) frameId = requestAnimationFrame(frame)
}

function scrollForPlayback(left) {
    viewport.value.scrollLeft = left
    playbackScrollLeft = viewport.value.scrollLeft
}

function drawSignature(signature) {
    if (signature.key === signatureKey) return
    signatureKey = signature.key
    signatureSurface.value.replaceChildren()
    // Use the same VexFlow stave and SVG context classes as the main OSMD
    // renderer. Symbols retain their original glyphs, sizes and staff heights.
    const Context = signature.staves[0].getContext().constructor
    const context = new Context(signatureSurface.value)
    context.resize(signatureWidth.value, svgHeight)
    context.setViewBox(0, cropTop, signatureWidth.value, svgHeight)
    context.setFillStyle('#161616')
    context.setStrokeStyle('#161616')
    context.setLineWidth(1)
    for (const stave of signature.staves) stave.setWidth(signatureWidth.value - stave.x + 1).setContext(context).draw()
}

function selectColumn(index, now) {
    const column = anchorPoints[index]
    if (activeColumn >= 0) for (const note of anchorPoints[activeColumn].notes) note.setColor('#161616', colorOptions)
    for (const note of column.notes) note.setColor('#b66b35', colorOptions)
    if (!browsing.value) drawSignature(column.signature)
    const absoluteX = column.x + musicStart.value - firstNoteX.value
    const currentX = absoluteX - pageOffset
    const snap = forcePosition || activeColumn < 0 || props.playerState.scrubbing
    if (browsing.value) {
        motion = null
        pageOffset = viewport.value.scrollLeft
        cursorLeft.value = absoluteX - pageOffset
    } else if (snap) {
        if (currentX < musicStart.value || currentX > props.width - 30) pageOffset = absoluteX - musicStart.value
        motion = null
        scrollForPlayback(pageOffset)
        cursorLeft.value = absoluteX - pageOffset
    } else {
        const nextPage = currentX > props.width - 30 || currentX < musicStart.value - 1
        const targetOffset = nextPage ? absoluteX - musicStart.value : pageOffset
        const gap = (anchorPoints[index + 1]?.ms ?? column.ms + 250) - column.ms
        const duration = Math.min(nextPage ? 220 : 120, Math.max(20, gap / (clock.state.speed || 1) * .65))
        motion = {at: now, duration, fromScroll: viewport.value.scrollLeft, toScroll: targetOffset,
            fromCursor: cursorLeft.value, toCursor: absoluteX - targetOffset}
        pageOffset = targetOffset
    }
    activeColumn = index
    forcePosition = false
}

function songMs(now) {
    const state = clock.state
    if (state.scrubbing || state.status !== 'playing') return clock.ms
    if (state.waiting) return state.currentStep?.ms ?? clock.ms
    const elapsed = Math.min(200, Math.max(0, now - clock.at)) * (state.speed || 1)
    if (state.leadInMs > 0) return clock.ms - Math.max(0, state.leadInMs - elapsed)
    let ms = clock.ms + elapsed
    if (state.mode === 'follow' && state.currentStep?.ms >= clock.ms) ms = Math.min(ms, state.currentStep.ms)
    ms = Math.min(ms, state.rightMs || props.timeline.durationMs)
    presentedMs = Math.max(presentedMs, ms)
    return presentedMs
}

function frame(now) {
    frameId = 0
    if (!mounted || !props.active || !layoutReady.value || !anchorPoints.length) return
    const ms = songMs(now)
    let low = 0, high = anchorPoints.length - 1
    while (low < high) {
        const middle = (low + high + 1) >> 1
        if (anchorPoints[middle].ms <= ms) low = middle
        else high = middle - 1
    }
    if (low < firstColumn) low = firstColumn
    if (low !== activeColumn || forcePosition) selectColumn(low, now)
    if (motion) {
        const progress = Math.min(1, (now - motion.at) / motion.duration)
        const eased = 1 - Math.pow(1 - progress, 3)
        scrollForPlayback(motion.fromScroll + (motion.toScroll - motion.fromScroll) * eased)
        cursorLeft.value = motion.fromCursor + (motion.toCursor - motion.fromCursor) * eased
        if (progress === 1) motion = null
    }
    if ((clock.state.status === 'playing' && !clock.state.waiting && !clock.state.scrubbing) || motion)
        frameId = requestAnimationFrame(frame)
}

function syncClock() {
    const state = props.playerState, now = performance.now()
    if (state.clockRevision !== clock.state.clockRevision || state.currentMs < clock.ms - 1 || state.scrubbing) {
        forcePosition = true
        presentedMs = state.currentMs
    }
    clock = {ms: state.currentMs || 0, at: state.receivedAt ?? now, state: {...state}}
    if (mounted && props.active && !frameId) frameId = requestAnimationFrame(frame)
}

function measurePositions(engine, samples, types) {
    const ordered = samples.slice().sort((a, b) => a.beat - b.beat || Number(b.attack) - Number(a.attack) || a.ms - b.ms)
    const beats = []
    for (const sample of ordered) if (!beats.length || sample.beat - beats.at(-1).beat > .00001) beats.push(sample)
    if (beats.length < 2) return []
    const svgLeft = scoreSurface.value.querySelector('svg').getBoundingClientRect().left
    const points = [], staffStates = new Map(), signatures = new Map()
    const clefs = [{1: 'french', 2: 'treble'}, {3: 'baritone-f', 4: 'bass', 5: 'subbass'},
        {1: 'soprano', 2: 'mezzo-soprano', 3: 'alto', 4: 'tenor', 5: 'baritone-c'}]
    let headerWidth = 120
    function readInstructions(state, instructions) {
        for (const instruction of instructions) {
            if (instruction instanceof types.KeyInstruction) state.fifths = instruction.Key
            if (instruction instanceof types.RhythmInstruction) state.time = instruction.SynthesizedFromNoTimeSignature ? null
                : instruction.SymbolEnum === 1 ? 'C' : instruction.SymbolEnum === 2 ? 'C|'
                    : `${instruction.Rhythm.Numerator}/${instruction.Rhythm.Denominator}`
            if (instruction instanceof types.ClefInstruction) {
                state.clef = instruction.ClefType < 3 ? clefs[instruction.ClefType][instruction.Line]
                    : instruction.ClefType === 3 ? 'percussion' : 'tab'
                state.annotation = instruction.OctaveOffset > 0 ? '8va' : instruction.OctaveOffset < 0 ? '8vb' : undefined
            }
        }
    }
    function signatureFor(measures) {
        const labels = measures.map(measure => staffStates.get(measure.ParentStaff.Id))
        const key = JSON.stringify(labels)
        if (!signatures.has(key)) {
            const staves = measures.map((measure, index) => {
                const source = measure.getVFStave(), label = labels[index]
                const stave = new source.constructor(12, source.y, 500, {...source.getOptions(), left_bar: false, right_bar: false})
                stave.addClef(label.clef, 'default', label.annotation)
                stave.addKeySignature(Key.majorTonicFromKeySignature(label.fifths))
                if (label.time) stave.addTimeSignature(label.time)
                headerWidth = Math.max(headerWidth, Math.ceil(stave.getNoteStartX() + 16))
                stave.setContext(source.getContext())
                return stave
            })
            signatures.set(key, {key, staves})
        }
        return signatures.get(key)
    }
    for (const measures of engine.GraphicSheet.MeasureList) {
        const visible = measures.filter(measure => measure.isVisible())
        if (!visible.length) continue
        for (const measure of visible) {
            const staff = measure.ParentStaff
            const state = {...(staffStates.get(staff.Id) || {fifths: 0, time: '4/4'})}
            const entry = measure.parentSourceMeasure.FirstInstructionsStaffEntries.find(item => item?.ParentStaff === staff)
            readInstructions(state, [measure.InitiallyActiveClef, ...(entry?.Instructions || [])])
            staffStates.set(staff.Id, state)
        }
        const entries = visible.flatMap(measure => measure.staffEntries.map(entry => ({measure, entry,
            beat: entry.getAbsoluteTimestamp().RealValue * 4}))).sort((a, b) => a.beat - b.beat)
        for (let index = 0; index < entries.length;) {
            let endIndex = index + 1
            while (endIndex < entries.length && Math.abs(entries[endIndex].beat - entries[index].beat) < .00001) endIndex++
            // Apply changes from both staves before constructing this column's
            // fixed header, including changes occurring inside a measure.
            for (let item = index; item < endIndex; item++) {
                const {measure, entry} = entries[item]
                readInstructions(staffStates.get(measure.ParentStaff.Id), entry.sourceStaffEntry?.Instructions || [])
            }
            const signature = signatureFor(visible)
            for (let item = index; item < endIndex; item++) {
                const {entry, beat} = entries[item]
                if (beat > beats.at(-1).beat + .00001) continue
                let low = 0, high = beats.length - 1
                while (low < high) {
                    const middle = (low + high + 1) >> 1
                    if (beats[middle].beat <= beat) low = middle
                    else high = middle - 1
                }
                const start = beats[low], end = beats[low + 1]
                const ms = end ? start.ms + Math.max(0, beat - start.beat) / (end.beat - start.beat) * (end.ms - start.ms) : start.ms
                const notes = entry.graphicalVoiceEntries.flatMap(voice => voice.notes).filter(note => !note.sourceNote.isRest())
                let left = Infinity, right = -Infinity
                for (const head of new Set(notes.flatMap(note => note.getNoteheadSVGs()))) {
                    const box = head.getBoundingClientRect()
                    left = Math.min(left, box.left)
                    right = Math.max(right, box.right)
                }
                const x = notes.length && Number.isFinite(left) ? (left + right) / 2 - svgLeft
                    : entry.PositionAndShape.AbsolutePosition.x * 10
                for (const note of notes) {
                    const element = note.getSVGGElement()
                    if (!element) continue
                    element.dataset.scoreMs = String(ms)
                    element.setAttribute('tabindex', '0')
                    element.setAttribute('role', 'button')
                    element.setAttribute('aria-label', `跳转到音符 ${note.sourceNote.NoteAsString}`)
                }
                points.push({beat, ms, x, notes, signature})
            }
            index = endIndex
        }
    }
    signatureWidth.value = headerWidth
    points.sort((a, b) => a.beat - b.beat || a.x - b.x)
    const columns = []
    for (const point of points) {
        const last = columns.at(-1)
        if (last && Math.abs(point.beat - last.beat) < .00001) {
            const count = last.notes.length + point.notes.length
            if (count) last.x = (last.x * last.notes.length + point.x * point.notes.length) / count
            last.notes.push(...point.notes)
        } else columns.push(point)
    }
    return columns
}

async function rebuild() {
    const revision = ++layoutRevision
    if (frameId) cancelAnimationFrame(frameId)
    frameId = 0
    if (!mounted || !props.active || !props.height || !props.width) return
    layoutReady.value = false
    pending.value = true
    error.value = ''
    anchorPoints = []
    activeColumn = -1
    motion = null
    signatureKey = ''
    pageOffset = 0
    playbackScrollLeft = 0
    forcePosition = true
    browsing.value = false
    dragging.value = false
    panGesture = null
    ignoreNextClick = false
    firstNoteX.value = 0
    viewport.value.scrollLeft = 0
    viewport.value.scrollTop = 0
    scoreSurface.value.style.width = `${props.width}px`
    const native = props.scoreDocument?.native
    const score = native ? {xml: props.scoreDocument.xml, positions: [
        ...(props.timeline?.notes || []).flatMap(note => [{beat: note.startBeat, ms: note.startMs, attack: true},
            {beat: note.endBeat, ms: note.endMs, attack: false}]),
        ...(props.timeline?.measures || []).map(mark => ({beat: mark.startBeat, ms: mark.ms, attack: false})),
    ]} : midiTimelineToScore(props.timeline)
    if (!score.xml) {
        pending.value = false
        error.value = '暂无可显示的乐谱'
        scoreSurface.value.replaceChildren()
        return
    }
    await nextTick()
    try {
        const types = await import('opensheetmusicdisplay')
        if (!mounted || revision !== layoutRevision) return
        scoreSurface.value.replaceChildren()
        const engine = new types.OpenSheetMusicDisplay(scoreSurface.value, {
            backend: 'svg', autoResize: false, renderSingleHorizontalStaffline: true,
            drawingParameters: 'default', drawTitle: false, drawSubtitle: false,
            drawComposer: false, drawCredits: false, drawPartNames: false,
            drawFingerings: !!native, drawMeasureNumbers: true, measureNumberInterval: 1,
            autoBeam: true, alignRests: 2, darkMode: false, coloringEnabled: false,
            defaultColorMusic: '#161616', defaultColorNotehead: '#161616',
            defaultColorStem: '#161616', defaultColorRest: '#161616', defaultColorLabel: '#444444',
            disableCursor: true, followCursor: false,
        })
        engine.EngravingRules.PageTopMargin = 1
        engine.EngravingRules.PageBottomMargin = 1
        engine.EngravingRules.StaffDistance = 5
        engine.EngravingRules.BetweenStaffDistance = 5
        engine.EngravingRules.SnapStafflinesToCrispPixels = false
        engine.zoom = 1
        await engine.load(score.xml)
        if (!mounted || revision !== layoutRevision) return
        const instruments = engine.Sheet.Instruments
        if (props.handMode !== 'both') {
            if (instruments.length === 1 && instruments[0].Staves.length === 2)
                instruments[0].Staves[props.handMode === 'left' ? 0 : 1].Visible = false
            else if (instruments.length === 2 && instruments.every(part => part.Staves.length === 1))
                instruments[props.handMode === 'left' ? 0 : 1].Visible = false
        }
        engine.render()
        const svg = scoreSurface.value.querySelector('svg'), bounds = svg.getBBox()
        cropTop = Math.floor(bounds.y) - 12
        svgHeight = Math.ceil(bounds.height) + 24
        // Crop unused page margins at 1:1 scale instead of shrinking glyphs.
        const viewBox = svg.viewBox.baseVal
        svg.setAttribute('viewBox', `${viewBox.x} ${cropTop} ${viewBox.width} ${svgHeight}`)
        svg.setAttribute('height', String(svgHeight))
        svg.style.height = `${svgHeight}px`
        sourceTop.value = Math.max(6, (props.height - svgHeight) / 2)
        scoreSurface.value.style.width = `${Math.ceil(svg.getBoundingClientRect().width + props.width)}px`
        anchorPoints = measurePositions(engine, score.positions, types)
        firstColumn = anchorPoints.findIndex(column => column.notes.length)
        if (firstColumn < 0) throw new Error('无法定位乐谱音符')
        firstNoteX.value = anchorPoints[firstColumn].x
        // The fixed header replaces the source's opening symbols. Hiding the
        // originals prevents fragments of time/key signatures leaking past it.
        const firstNote = svg.getBoundingClientRect().left + firstNoteX.value
        for (const symbol of svg.querySelectorAll('.vf-clef, .vf-keysignature, .vf-timesignature')) {
            if (symbol.getBoundingClientRect().right <= firstNote) symbol.style.visibility = 'hidden'
        }
        drawSignature(anchorPoints[firstColumn].signature)
        layoutReady.value = true
        pending.value = false
        syncClock()
    } catch (failure) {
        if (!mounted || revision !== layoutRevision) return
        console.error('五线谱排版失败', failure)
        pending.value = false
        error.value = '乐谱排版失败，请重新选择曲目'
        scoreSurface.value.replaceChildren()
    }
}
watch(() => props.playerState, syncClock)
watch(() => [props.scoreDocument, props.timeline, props.handMode, Math.floor(props.height / 16), props.active], rebuild)
watch(() => props.width, () => {forcePosition = true; syncClock()})
onMounted(() => {mounted = true; void rebuild()})
onBeforeUnmount(() => {mounted = false; panGesture = null; layoutRevision++; if (frameId) cancelAnimationFrame(frameId); anchorPoints = []})
</script>

<style scoped>
.score-panel { position: relative; overflow: hidden; background: #fff; color: #161616; border-bottom: 1px solid #dedbd5; --wails-draggable: no-drag; }
.engraving { position: absolute; inset: 0; overflow: auto; overscroll-behavior: contain; scrollbar-width: thin; scrollbar-color: #c8c5bf #f8f7f4; touch-action: pan-y; cursor: grab; }
.score-surface { position: absolute; top: 0; left: 0; width: 100%; }
.score-surface :deep(svg), .signature-surface :deep(svg) { display: block; max-width: none !important; shape-rendering: geometricPrecision; text-rendering: geometricPrecision; }
.score-surface :deep([data-score-ms]) { cursor: pointer; }
.score-surface :deep([data-score-ms]:focus-visible) { outline: 1px solid #b66b35; outline-offset: 3px; }
.engraving.is-dragging, .engraving.is-dragging :deep(*) { cursor: grabbing !important; user-select: none; }
.signature-panel { position: absolute; z-index: 4; left: 0; top: 0; bottom: 10px; overflow: hidden; background: #fff; }
.signature-panel::after { content: ''; position: absolute; right: 0; top: 12px; bottom: 12px; width: 1px; background: #00000010; }
.resume-follow { position: absolute; z-index: 5; right: 14px; top: 10px; padding: 6px 10px; border: 1px solid #ddd6cc; border-radius: 5px; background: #fffffff0; color: #775536; font: inherit; font-size: 12px; cursor: pointer; }
.resume-follow:focus-visible { outline: 2px solid #b66b35; outline-offset: 2px; }
.signature-surface { position: absolute; left: 0; width: 100%; }
.playhead { position: absolute; z-index: 3; top: 12px; bottom: 20px; width: 28px; transform: translateX(-14px); border-radius: 3px; background: #d08b3c26; border: 1px solid #c9893a42; pointer-events: none; }
.playhead.is-waiting { background: #d08b3c38; }
.score-caption { position: absolute; z-index: 4; right: 12px; bottom: 13px; font-size: 10px; line-height: 1.4; color: #8b8882; background: #ffffffe8; }
.score-message { position: absolute; inset: 0; display: grid; place-items: center; margin: 0; font-size: 13px; color: #777; }
</style>
