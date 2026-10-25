export const paletteGroups = ['清晰练习', '柔和耐看', '演奏展示', '经典质感']

function palette(key, label, group, desc, right, rightBlack, left, leftBlack) {
    return {key, label, group, desc, recommended: key === 'blue-amber', colors: {
        whiteKey: right, blackKey: rightBlack, whiteKeyLeft: left, blackKeyLeft: leftBlack,
        damperPedal: '#D3AD68', softPedal: '#87BEA5', sostenutoPedal: '#8EAFE0',
    }}
}

export const colorPalettes = [
    palette('blue-amber', '蓝橙练习', '清晰练习', '柔蓝与琥珀，冷暖分明，优先推荐。', '#8DB8DF', '#6095C5', '#E1B881', '#BD8E55'),
    palette('teal-coral', '冰青珊瑚', '清晰练习', '冰青与珊瑚，清爽、声部清楚。', '#85C8C1', '#57A39B', '#E0A28F', '#BC7B68'),
    palette('apple-blue', '苹果蓝紫', '清晰练习', '蓝紫色调，柔化高亮，保留冷色层次。', '#88B5E5', '#5F8BC5', '#B7A7DF', '#8E7ABB'),
    palette('indigo-gold', '靛蓝金砂', '清晰练习', '靛蓝与金砂，稳重的冷暖对照。', '#9EABE1', '#7887BE', '#D9BD78', '#B6974F'),
    palette('fresh', '清透薄荷', '柔和耐看', '薄荷与琥珀，收敛荧光感，清爽柔和。', '#91D1AE', '#64AB86', '#E5BE84', '#BF955D'),
    palette('soft-pro', '低饱和专业', '柔和耐看', '灰绿与淡紫，克制的练习配色。', '#A5C1B2', '#7D9D8C', '#BCB5CC', '#928AA7'),
    palette('mist-almond', '雾蓝杏仁', '柔和耐看', '柔和雾蓝与暖杏，安静耐看。', '#AABFD0', '#809AAE', '#D5BDA4', '#B2977C'),
    palette('sage-clay', '鼠尾草陶土', '柔和耐看', '鼠尾草绿与陶土，自然温和。', '#AEC2A4', '#829B76', '#CCA18A', '#AA7B64'),
    palette('lilac-peach', '丁香蜜桃', '柔和耐看', '淡紫与蜜桃，轻柔的暖冷搭配。', '#BDAED6', '#9786B4', '#DFB69D', '#BC9076'),
    palette('warm-stage', '暖色舞台', '演奏展示', '暖橙与冷蓝，温暖又能分清左右手。', '#EBA779', '#C78052', '#94BCCB', '#6797AA'),
    palette('neon-practice', '霓虹练习', '演奏展示', '鲜明青紫，适合演示与快速观察。', '#58CFE1', '#2CA7BF', '#D18ACC', '#AD5EAC'),
    palette('sakura-ice', '樱粉冰青', '演奏展示', '樱粉与冰青，明快而有辨识度。', '#E0A0B8', '#BD7792', '#93CCD5', '#66A6B2'),
    palette('twilight-rose', '暮紫玫瑰', '演奏展示', '暮紫与玫瑰，柔和的舞台氛围。', '#AAA2D8', '#8077B5', '#D69CAE', '#B4758A'),
    palette('classic-ivory', '古典象牙', '经典质感', '象牙金与灰绿，加深激活色以便辨认。', '#D2B684', '#AD8D56', '#9FB9A8', '#6F9581'),
    palette('studio-focus', '录音棚聚焦', '经典质感', '冷灰蓝与暖灰，简洁、专注。', '#B7C1CE', '#8998AA', '#CCB69B', '#A58B6D'),
    palette('silver-copper', '银灰铜金', '经典质感', '银灰与铜金，稳重的金属色搭配。', '#C1C8D0', '#929EAB', '#CFA879', '#AB8359'),
]
