const DEFAULT_ACCENT = '#9af7b3'
const CONTROL_SURFACE = [42, 42, 42]

function parseColor(value) {
    const match = String(value || '').trim().match(/^#([\da-f]{3}|[\da-f]{4}|[\da-f]{6}|[\da-f]{8})$/i)
    if (!match) return null
    const hex = match[1].length < 5
        ? match[1].slice(0, 3).split('').map(char => char + char).join('')
        : match[1].slice(0, 6)
    return [0, 2, 4].map(offset => parseInt(hex.slice(offset, offset + 2), 16))
}

function luminance(rgb) {
    return rgb.map(channel => {
        const value = channel / 255
        return value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4
    }).reduce((sum, value, index) => sum + value * [0.2126, 0.7152, 0.0722][index], 0)
}

function contrast(a, b) {
    const first = luminance(a)
    const second = luminance(b)
    return (Math.max(first, second) + 0.05) / (Math.min(first, second) + 0.05)
}

function mix(rgb, target, amount) {
    return rgb.map((channel, index) => Math.round(channel + (target[index] - channel) * amount))
}

function readableAccent(rgb, appearance = 'dark') {
    const surface = appearance === 'light' ? [255, 255, 255] : CONTROL_SURFACE
    const target = appearance === 'light' ? [0, 0, 0] : [255, 255, 255]
    if (contrast(rgb, surface) >= 4.5) return rgb
    for (let amount = 0.02; amount <= 1; amount += 0.02) {
        const adjusted = mix(rgb, target, amount)
        if (contrast(adjusted, surface) >= 4.5) return adjusted
    }
    return target
}

const hexColor = rgb => '#' + rgb.map(channel => channel.toString(16).padStart(2, '0')).join('')
const alphaColor = (rgb, alpha) => `rgba(${rgb.join(', ')}, ${alpha})`

// Neutral surfaces stay gray. Only emphasis and interaction states inherit the
// existing piano key color, adjusted for legibility against the active surface.
export function getPianoUiTheme(colors = {}, appearance = 'dark') {
    const whiteKey = colors?.whiteKey
    const inputColor = typeof whiteKey === 'string' ? whiteKey : whiteKey?.color
    const isLight = appearance === 'light'
    const rgb = readableAccent(parseColor(inputColor) || parseColor(DEFAULT_ACCENT), appearance)
    const accent = hexColor(rgb)
    const accentHover = hexColor(mix(rgb, isLight ? [0, 0, 0] : [255, 255, 255], 0.12))
    const accentPressed = hexColor(readableAccent(mix(rgb, [0, 0, 0], isLight ? 0.22 : 0.12), appearance))
    const accentText = contrast(rgb, [23, 23, 23]) >= contrast(rgb, [255, 255, 255]) ? '#171717' : '#ffffff'
    const accentSoft = alphaColor(rgb, 0.12)
    const accentRing = alphaColor(rgb, 0.25)
    const focusRing = `0 0 0 2px ${accentRing}`

    const naiveOverrides = {
        common: {
            primaryColor: accent,
            primaryColorHover: accentHover,
            primaryColorPressed: accentPressed,
            primaryColorSuppl: accent,
            bodyColor: '#1f1f1f',
            cardColor: '#2a2a2a',
            modalColor: '#303030',
            popoverColor: '#333333',
            tableColor: '#2a2a2a',
            inputColor: '#363636',
            inputColorDisabled: '#2e2e2e',
            actionColor: '#303030',
            tableHeaderColor: '#303030',
            tabColor: '#303030',
            tagColor: '#3a3a3a',
            railColor: '#555555',
            borderColor: 'rgba(255,255,255,.16)',
            dividerColor: 'rgba(255,255,255,.09)',
            hoverColor: 'rgba(255,255,255,.06)',
            pressedColor: 'rgba(255,255,255,.04)',
            textColor1: '#eeeeee',
            textColor2: '#d7d7d7',
            textColor3: '#999999',
            borderRadius: '6px',
        },
        Card: {color: '#2a2a2a', colorModal: '#303030', colorPopover: '#333333', colorEmbedded: '#303030'},
        Popover: {color: '#333333', textColor: '#eeeeee'},
        Select: {
            peers: {
                InternalSelection: {
                    color: '#363636',
                    colorActive: '#363636',
                    colorDisabled: '#2e2e2e',
                    border: '1px solid rgba(255,255,255,.14)',
                    borderHover: `1px solid ${accentHover}`,
                    borderActive: `1px solid ${accent}`,
                    borderFocus: `1px solid ${accent}`,
                    boxShadowActive: focusRing,
                    boxShadowFocus: focusRing,
                    caretColor: accent,
                    loadingColor: accent,
                },
                InternalSelectMenu: {
                    color: '#333333',
                    height: 'min(50vh, 240px)',
                    optionFontSizeSmall: '12px',
                    optionFontSizeMedium: '12px',
                    optionHeightSmall: '30px',
                    optionHeightMedium: '30px',
                    optionTextColorActive: accent,
                    optionTextColorPressed: accentPressed,
                    optionCheckColor: accent,
                    optionColorPending: 'rgba(255,255,255,.06)',
                    optionColorActive: accentSoft,
                    optionColorActivePending: accentSoft,
                },
            },
        },
        Slider: {
            fillColor: accent,
            fillColorHover: accentHover,
            railColor: '#555555',
            railColorHover: '#626262',
            railHeight: '3px',
            handleSize: '12px',
            dotHeight: '2px',
            dotWidth: '2px',
            fontSize: '12px',
            markFontSize: '12px',
            handleBoxShadowFocus: focusRing,
        },
        Switch: {railColor: '#555555', railColorActive: accent, boxShadowFocus: focusRing},
        Radio: {
            fontSizeSmall: '12px',
            fontSizeMedium: '12px',
            buttonHeightSmall: '28px',
            buttonHeightMedium: '28px',
            buttonBorderRadius: '3px',
            buttonBorderColorActive: accent,
            buttonBorderColorHover: accentHover,
            buttonColor: '#363636',
            buttonColorActive: accent,
            buttonTextColorActive: accentText,
            buttonTextColorHover: accentHover,
            buttonBoxShadowFocus: focusRing,
        },
        Button: {
            textColorPrimary: accentText,
            textColorHoverPrimary: accentText,
            textColorPressedPrimary: accentText,
            textColorFocusPrimary: accentText,
        },
    }

    return {
        accent,
        accentHover,
        accentPressed,
        accentText,
        naiveOverrides,
        cssVariables: {
            '--ui-accent': accent,
            '--ui-accent-hover': accentHover,
            '--ui-accent-pressed': accentPressed,
            '--ui-accent-text': accentText,
            '--ui-accent-soft': accentSoft,
            '--ui-accent-ring': accentRing,
        },
    }
}
