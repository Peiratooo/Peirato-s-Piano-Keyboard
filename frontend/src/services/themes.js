export const colorPalettes = [
    {
        key: 'fresh',
        label: '清透薄荷',
        desc: '清爽明亮，接近默认风格。',
        preview: ['#9AF7B3', '#5FFF5F', '#f7e89a', '#ffd25f', '#10e786'],
        colors: {
            whiteKey: '#9AF7B3',
            blackKey: '#5FFF5F',
            whiteKeyLeft: '#f7e89a',
            blackKeyLeft: '#ffd25f',
            damperPedal: '#e7b510',
            softPedal: '#10e786',
            sostenutoPedal: '#1054e7',
        },
    },
    {
        key: 'apple-blue',
        label: '苹果蓝紫',
        desc: '现代冷色，高亮清晰。',
        preview: ['#7DD3FC', '#38BDF8', '#C4B5FD', '#8B5CF6', '#34D399'],
        colors: {
            whiteKey: '#7DD3FC',
            blackKey: '#38BDF8',
            whiteKeyLeft: '#C4B5FD',
            blackKeyLeft: '#8B5CF6',
            damperPedal: '#F59E0B',
            softPedal: '#34D399',
            sostenutoPedal: '#60A5FA',
        },
    },
    {
        key: 'warm-stage',
        label: '暖色舞台',
        desc: '偏演奏氛围，温暖。',
        preview: ['#FDBA74', '#FB923C', '#FDE68A', '#F59E0B', '#A78BFA'],
        colors: {
            whiteKey: '#FDBA74',
            blackKey: '#FB923C',
            whiteKeyLeft: '#FDE68A',
            blackKeyLeft: '#F59E0B',
            damperPedal: '#FACC15',
            softPedal: '#4ADE80',
            sostenutoPedal: '#A78BFA',
        },
    },
    {
        key: 'soft-pro',
        label: '低饱和专业',
        desc: '更克制，适合长时间练习。',
        preview: ['#A7F3D0', '#6EE7B7', '#BFDBFE', '#93C5FD', '#FCD34D'],
        colors: {
            whiteKey: '#A7F3D0',
            blackKey: '#6EE7B7',
            whiteKeyLeft: '#BFDBFE',
            blackKeyLeft: '#93C5FD',
            damperPedal: '#FCD34D',
            softPedal: '#86EFAC',
            sostenutoPedal: '#93C5FD',
        },
    },
    {
        key: 'neon-practice',
        label: '霓虹练习',
        desc: '对比更强，适合快速观察按键反馈。',
        preview: ['#22D3EE', '#06B6D4', '#F0ABFC', '#D946EF', '#FDE047'],
        colors: {
            whiteKey: '#22D3EE',
            blackKey: '#06B6D4',
            whiteKeyLeft: '#F0ABFC',
            blackKeyLeft: '#D946EF',
            damperPedal: '#FDE047',
            softPedal: '#4ADE80',
            sostenutoPedal: '#60A5FA',
        },
    },
    {
        key: 'classic-ivory',
        label: '古典象牙',
        desc: '柔和暖白，适合浅色界面。',
        preview: ['#FDECC8', '#E9B872', '#D8F3DC', '#95D5B2', '#8ECAE6'],
        colors: {
            whiteKey: '#FDECC8',
            blackKey: '#E9B872',
            whiteKeyLeft: '#D8F3DC',
            blackKeyLeft: '#95D5B2',
            damperPedal: '#FFB703',
            softPedal: '#52B788',
            sostenutoPedal: '#8ECAE6',
        },
    },
    {
        key: 'studio-focus',
        label: '录音棚聚焦',
        desc: '灰蓝基调，状态色更稳重。',
        preview: ['#CBD5E1', '#94A3B8', '#BAE6FD', '#38BDF8', '#FBBF24'],
        colors: {
            whiteKey: '#CBD5E1',
            blackKey: '#94A3B8',
            whiteKeyLeft: '#BAE6FD',
            blackKeyLeft: '#38BDF8',
            damperPedal: '#FBBF24',
            softPedal: '#6EE7B7',
            sostenutoPedal: '#93C5FD',
        },
    },
]


export function applyColorPalette(config, key) {
    const palette = colorPalettes.find(item => item.key === key)
    if (!palette) return false
    for (const [name, color] of Object.entries(palette.colors)) {
        if (config.colors[name]) config.colors[name] = {...config.colors[name], color}
    }
    return true
}
