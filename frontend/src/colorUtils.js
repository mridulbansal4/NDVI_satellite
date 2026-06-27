/**
 * colorUtils.js
 * Palette mapping and conversion utilities.
 */

// EOS-style diverging palette: Dark Red (bad) -> Yellow -> Dark Green (good)
const EOS_PALETTE = [
    '#ad0028', '#c5142a', '#e02d2c', '#ef4c3a', '#fe6c4a', 
    '#ff8d5a', '#ffab69', '#ffc67d', '#ffe093', '#ffefab', 
    '#fdfec2', '#eaf7ac', '#d5ef94', '#b9e383', '#9bd873', 
    '#77ca6f', '#53bd6b', '#14aa60', '#009755', '#007e47', '#007e47'
];

/** Convert NDVI/CVI/EVI value (-1 to 1) to a hex color using EOS palette */
export function ndviToColor(value) {
  if (value === null || value === undefined) return '#4b5563'; // fallback grey
  
  // Values below 0.0 usually mean no vegetation (water, bare rock)
  if (value <= 0) return EOS_PALETTE[0];
  if (value >= 1) return EOS_PALETTE[EOS_PALETTE.length - 1];

  const scaled = value * (EOS_PALETTE.length - 1);
  const idx = Math.floor(scaled);

  return EOS_PALETTE[idx] || "#007f00";
}

/** Convert "rgb(r,g,b)" and "#RRGGBB" -> "rgba(r,g,b,alpha)" for canvas gradient stops */
export function _rgba(colorStr, alpha) {
  if (colorStr.startsWith('#')) {
    const c = colorStr.substring(1).split('');
    let hex = c;
    if (c.length === 3) {
      hex = [c[0], c[0], c[1], c[1], c[2], c[2]];
    }
    hex = hex.join('');
    const r = parseInt(hex.slice(0, 2), 16);
    const g = parseInt(hex.slice(2, 4), 16);
    const b = parseInt(hex.slice(4, 6), 16);
    return `rgba(${r},${g},${b},${alpha})`;
  }
  return colorStr.replace('rgb(', 'rgba(').replace(')', `,${alpha})`);
}
