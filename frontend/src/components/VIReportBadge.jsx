/**
 * VIReportBadge.jsx
 * Props: cvi_mean (number 0–1)
 * Renders a color-coded health label pill. No emojis.
 */

export default function VIReportBadge({ cvi_mean }) {
  if (cvi_mean == null) {
    return (
      <span className="text-[10px] font-bold px-2.5 py-1 rounded border bg-[#1a2432] text-[#7a90a8] border-[#ffffff1a]">
        No Data
      </span>
    );
  }

  const cfg =
    cvi_mean >= 0.6
      ? { label: 'Healthy',  bg: '#f0fdf4', color: '#166534', border: '#bbf7d0' }
      : cvi_mean >= 0.4
      ? { label: 'Moderate', bg: '#fffbeb', color: '#92400e', border: '#fde68a' }
      : { label: 'Stressed', bg: '#fef2f2', color: '#991b1b', border: '#fecaca' };

  return (
    <span
      className="text-[10px] font-bold px-2.5 py-1 rounded border"
      style={{ backgroundColor: cfg.bg, color: cfg.color, borderColor: cfg.border }}
    >
      {cfg.label}
    </span>
  );
}
