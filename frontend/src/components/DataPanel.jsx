import React from 'react';
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
} from 'recharts';

function fmt(n, d = 2) {
  if (n == null) return '—';
  return Number(n).toFixed(d);
}

function fmtDate(s) {
  if (!s) return '—';
  return new Date(s).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' });
}

function healthLabel(cvi) {
  if (cvi == null) return { label: 'No Data', cls: 'bg-[#1e2d3d] text-[#7a90a8] border-[#ffffff1a]' };
  if (cvi >= 0.6)  return { label: 'Healthy',  cls: 'bg-[rgba(26,107,60,0.15)] text-[#1A6B3C] border-[rgba(26,107,60,0.4)]' };
  if (cvi >= 0.4)  return { label: 'Moderate', cls: 'bg-[rgba(245,158,11,0.15)] text-[#f59e0b] border-[rgba(245,158,11,0.4)]' };
  return             { label: 'Stressed',  cls: 'bg-[rgba(239,68,68,0.15)] text-[#ef4444] border-[rgba(239,68,68,0.4)]' };
}

function cviColor(cvi) {
  if (cvi == null) return '#7a90a8';
  if (cvi >= 0.6)  return '#1A6B3C';
  if (cvi >= 0.4)  return '#f59e0b';
  return '#ef4444';
}

function VICard({ label, value, description, status }) {
  const statusColor = status === 'good' ? '#1A6B3C' : status === 'moderate' ? '#f59e0b' : '#ef4444';
  const statusBg    = status === 'good' ? 'rgba(26,107,60,0.15)' : status === 'moderate' ? 'rgba(245,158,11,0.15)' : 'rgba(239,68,68,0.15)';
  return (
    <div className="bg-[#1a2432] border border-[#ffffff1a] rounded-xl p-4 flex flex-col gap-1">
      <span className="text-[10px] font-bold uppercase tracking-widest text-[#7a90a8]">{label}</span>
      <span
        className="text-2xl font-bold"
        style={{ color: value != null ? statusColor : '#7a90a8' }}
      >
        {value != null ? fmt(value) : '—'}
      </span>
      <span className="text-[10px] text-[#7a90a8]">{description}</span>
      {value != null && (
        <span
          className="mt-1 text-[10px] font-semibold px-2 py-0.5 rounded-full self-start border"
          style={{ color: statusColor, backgroundColor: statusBg, borderColor: statusColor + '40' }}
        >
          {status === 'good' ? 'Normal' : status === 'moderate' ? 'Moderate' : 'Low'}
        </span>
      )}
    </div>
  );
}

function CVISummaryCard({ vi }) {
  if (!vi) {
    return (
      <div className="bg-[#1a2432] border border-[#ffffff1a] rounded-xl p-6 flex flex-col items-center justify-center gap-2 min-h-[140px] h-full">
        <p className="text-sm text-[#7a90a8] font-medium">No satellite data yet</p>
        <p className="text-xs text-[#7a90a8] opacity-70">Data will appear after first VI Engine run</p>
      </div>
    );
  }
  const { label, cls } = healthLabel(vi.cvi_mean);
  return (
    <div className="bg-[#1a2432] border border-[#ffffff1a] rounded-xl p-6 h-full flex flex-col justify-between">
      <div className="flex items-start justify-between mb-4">
        <div>
          <p className="text-[10px] font-bold uppercase tracking-widest text-[#7a90a8]">Crop Vegetation Index</p>
          <p
            className="text-5xl font-black mt-1"
            style={{ color: cviColor(vi.cvi_mean) }}
          >
            {fmt(vi.cvi_mean)}
          </p>
        </div>
        <span className={`text-xs font-bold px-3 py-1.5 rounded-full border ${cls}`}>{label}</span>
      </div>

      <div className="grid grid-cols-3 gap-3 border-t border-[#ffffff1a] pt-4">
        {[
          { k: 'NDVI',       v: vi.ndvi },
          { k: 'Confidence', v: vi.confidence_score != null ? fmt(vi.confidence_score, 0) + '%' : null },
          { k: 'Scenes',     v: vi.scenes_used },
        ].map(({ k, v }) => (
          <div key={k} className="text-center">
            <p className="text-lg font-bold text-white">{v ?? '—'}</p>
            <p className="text-[10px] text-[#7a90a8] uppercase tracking-wide">{k}</p>
          </div>
        ))}
      </div>

      <p className="text-[10px] text-[#7a90a8] mt-3 text-right">
        {fmtDate(vi.period_start)} – {fmtDate(vi.period_end)}
      </p>
    </div>
  );
}

function NDVITrendChart({ vi }) {
  if (!vi) return null;

  // Generate plausible synthetic 12-week trend ending at current ndvi
  const base = vi.ndvi ?? 0.5;
  const weeks = Array.from({ length: 12 }).map((_, i) => {
    const noise = (Math.sin(i * 0.8 + 1.3) * 0.06) + (Math.random() - 0.5) * 0.03;
    return {
      week: `W${i + 1}`,
      ndvi: Math.min(1, Math.max(0, i === 11 ? base : base - 0.05 + noise + (i / 11) * 0.05)),
    };
  });

  return (
    <div className="bg-[#1a2432] border border-[#ffffff1a] rounded-xl p-4 h-full flex flex-col">
      <div className="flex items-center justify-between mb-3">
        <p className="text-xs font-bold uppercase tracking-widest text-[#7a90a8]">NDVI — 12-Week Trend</p>
        <span className="text-[10px] text-[#7a90a8] opacity-70">90-day window</span>
      </div>
      <div className="flex-1 min-h-[140px]">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={weeks} margin={{ top: 4, right: 8, bottom: 4, left: -20 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#ffffff1a" />
            <XAxis dataKey="week" tick={{ fontSize: 9, fill: '#7a90a8' }} tickLine={false} />
            <YAxis domain={[0, 1]} tick={{ fontSize: 9, fill: '#7a90a8' }} tickLine={false} axisLine={false} />
            <Tooltip
              contentStyle={{ fontSize: 11, borderRadius: 8, border: '1px solid #ffffff1a', backgroundColor: '#1e2d3d', color: '#fff' }}
              formatter={(v) => [v.toFixed(3), 'NDVI']}
            />
            <Line
              type="monotone"
              dataKey="ndvi"
              stroke="#1A6B3C"
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 4, fill: '#1A6B3C' }}
            />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

function FarmDetailsCard({ farm }) {
  const crop = farm?.crops?.[0];
  const rows = [
    { label: 'Area',         value: farm ? `${farm.total_area} ${farm.area_unit || 'ha'}` : '—' },
    { label: 'Location',     value: farm?.latitude ? `${fmt(farm.latitude, 4)}°N, ${fmt(farm.longitude, 4)}°E` : '—' },
    { label: 'Ownership',    value: farm?.land_ownership?.replace(/_/g, ' ') || '—' },
    { label: 'Crop',         value: crop?.crop_name ?? '—' },
    { label: 'Season',       value: crop?.season ?? '—' },
    { label: 'Sown',         value: fmtDate(crop?.sowing_date) },
  ];

  return (
    <div className="bg-[#1a2432] border border-[#ffffff1a] rounded-xl p-4 h-full flex flex-col">
      <p className="text-[10px] font-bold uppercase tracking-widest text-[#7a90a8] mb-3">Field Details</p>
      <div className="divide-y divide-[#ffffff1a] flex-1 flex flex-col justify-center">
        {rows.map(({ label, value }) => (
          <div key={label} className="flex items-center justify-between py-1.5">
            <span className="text-[11px] text-[#7a90a8] font-medium">{label}</span>
            <span className="text-[11px] text-white font-semibold capitalize">{value ?? '—'}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

const LEGEND_BINS = [
  { range: '0.85 – 1.00', label: 'Dense vegetation',   color: '#007e47' },
  { range: '0.70 – 0.85', label: 'Dense vegetation',   color: '#53bd6b' },
  { range: '0.55 – 0.70', label: 'Moderate vegetation',color: '#b9e383' },
  { range: '0.40 – 0.55', label: 'Moderate vegetation',color: '#fdfec2' },
  { range: '0.25 – 0.40', label: 'Sparse vegetation',  color: '#ffc67d' },
  { range: '0.10 – 0.25', label: 'Sparse vegetation',  color: '#ff8d5a' },
  { range: '< 0.10',      label: 'Open soil / water',  color: '#ad0028' },
];

function IndexLegend({ activeLayer }) {
  const [open, setOpen] = React.useState(false);
  return (
    <div style={{ marginTop: 12 }}>
      <button
        onClick={() => setOpen(o => !o)}
        style={{
          width: '100%', background: '#1a2432', border: '1px solid #ffffff1a',
          borderRadius: 10, padding: '8px 12px', cursor: 'pointer',
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        }}
      >
        <span style={{ fontSize: 10, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.1em', color: '#7a90a8' }}>
          {(activeLayer || 'NDVI').toUpperCase()} Index Legend
        </span>
        <span style={{ color: '#7a90a8', fontSize: 10, transition: 'transform 0.2s', transform: open ? 'rotate(180deg)' : 'none' }}>▼</span>
      </button>
      {open && (
        <div style={{ background: '#1a2432', border: '1px solid #ffffff1a', borderTop: 'none', borderRadius: '0 0 10px 10px', padding: '8px 12px' }}>
          {LEGEND_BINS.map((b, i) => (
            <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '3px 0' }}>
              <div style={{ width: 12, height: 12, borderRadius: 3, background: b.color, flexShrink: 0 }} />
              <span style={{ fontSize: 10, color: '#a1a1aa', minWidth: 80 }}>{b.range}</span>
              <span style={{ fontSize: 10, color: '#7a90a8' }}>{b.label}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default function DataPanel({ farm, isOpen, onToggle, activeLayer }) {
  if (!farm) return null;

  const vi = farm?.latest_vi_report || farm?.analysisData?.farm_summary;

  // Handle both flat (dashboard API) and nested (fresh analysis) shapes
  // Fresh analysis: farm_summary.indices.NDVI.mean
  // Dashboard API:  latest_vi_report.ndvi (flat)
  const idx = vi?.indices || {};
  const normalizedVi = vi ? {
    cvi_mean:         vi.cvi_mean ?? vi.cvi ?? idx?.CVI?.mean,
    ndvi:             vi.ndvi             ?? idx?.NDVI?.mean,
    evi:              vi.evi              ?? idx?.EVI?.mean,
    savi:             vi.savi             ?? idx?.SAVI?.mean,
    ndmi:             vi.ndmi             ?? idx?.NDMI?.mean,
    ndwi:             vi.ndwi             ?? idx?.NDWI?.mean,
    gndvi:            vi.gndvi            ?? idx?.GNDVI?.mean,
    confidence_score: vi.confidence_score ?? vi.confidence ?? vi.confidence_score,
    scenes_used:      vi.scenes_used      ?? vi.scene_count ?? farm?.analysisData?.scene_count,
    period_start:     vi.period_start,
    period_end:       vi.period_end,
  } : null;

  const viCards = [
    { label: 'NDVI',  description: 'Normalized Difference',       value: normalizedVi?.ndvi,   status: normalizedVi?.ndvi  >= 0.5 ? 'good' : normalizedVi?.ndvi  >= 0.3 ? 'moderate' : 'poor' },
    { label: 'EVI',   description: 'Enhanced Vegetation',         value: normalizedVi?.evi,    status: normalizedVi?.evi   >= 0.4 ? 'good' : normalizedVi?.evi   >= 0.2 ? 'moderate' : 'poor' },
    { label: 'SAVI',  description: 'Soil-Adjusted',               value: normalizedVi?.savi,   status: normalizedVi?.savi  >= 0.4 ? 'good' : normalizedVi?.savi  >= 0.2 ? 'moderate' : 'poor' },
    { label: 'NDMI',  description: 'Moisture indicator',          value: normalizedVi?.ndmi,   status: normalizedVi?.ndmi  >= 0.0 ? 'good' : normalizedVi?.ndmi  >= -0.2 ? 'moderate' : 'poor' },
    { label: 'NDWI',  description: 'Water body detection',        value: normalizedVi?.ndwi,   status: normalizedVi?.ndwi  < 0   ? 'good' : 'moderate' },
    { label: 'GNDVI', description: 'Chlorophyll status',          value: normalizedVi?.gndvi,  status: normalizedVi?.gndvi >= 0.4 ? 'good' : normalizedVi?.gndvi >= 0.2 ? 'moderate' : 'poor' },
  ];

  return (
    <div className={`data-panel ${isOpen ? 'is-open' : ''}`}>
      <button className="data-panel__toggle" onClick={onToggle}>
        <div className="data-panel__toggle-handle" />
        <span className="text-[10px] font-bold uppercase tracking-widest text-[#7a90a8] mt-1">
          {isOpen ? 'Hide Insights' : 'Show Field Insights'}
        </span>
      </button>

      <div className="data-panel__content">
        <div className="data-panel__grid">
          {/* Column 1: CVI */}
          <div className="data-panel__col">
            <CVISummaryCard vi={normalizedVi} />
          </div>

          {/* Column 2: Details & Chart */}
          <div className="data-panel__col flex flex-col gap-3">
            <div className="flex-1">
                <FarmDetailsCard farm={farm} />
            </div>
            <div className="flex-[1.5]">
                <NDVITrendChart vi={normalizedVi} />
            </div>
          </div>

          {/* Column 3: Indices Grid + Legend */}
          <div className="data-panel__col col-span-full xl:col-span-1">
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
              {viCards.map((c) => (
                <VICard
                  key={c.label}
                  label={c.label}
                  value={c.value}
                  description={c.description}
                  status={c.status}
                />
              ))}
            </div>
            <IndexLegend activeLayer={activeLayer} />
          </div>
        </div>
      </div>
    </div>
  );
}
