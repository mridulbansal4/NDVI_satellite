/**
 * FarmCard.jsx
 * Used in Dashboard sidebar farm list. No emojis. PRAGYA color theme.
 * Props: farm { id, farm_name, total_area, area_unit, crops, latest_vi_report }
 */

import VIReportBadge from './VIReportBadge';

function formatDate(s) {
  if (!s) return '—';
  return new Date(s).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' });
}

function SeasonPill({ season }) {
  const map = {
    kharif: 'bg-blue-50 text-blue-700 border-blue-200',
    rabi:   'bg-orange-50 text-orange-700 border-orange-200',
    zaid:   'bg-purple-50 text-purple-700 border-purple-200',
  };
  return (
    <span className={`text-[10px] font-semibold px-2 py-0.5 rounded border ${map[season] || 'bg-[#1a2432] text-[#7a90a8] border-[#ffffff1a]'}`}>
      {season?.charAt(0).toUpperCase() + season?.slice(1)}
    </span>
  );
}

export default function FarmCard({ farm }) {
  const vi   = farm.latest_vi_report;
  const crop = farm.crops?.[0];

  return (
    <div className="bg-[#111820] rounded-xl border border-[#ffffff1a] overflow-hidden hover:border-[#1A6B3C]/50 transition-colors">
      {/* Header — left green border accent */}
      <div className="border-l-4 px-4 py-3" style={{ borderLeftColor: '#1A6B3C' }}>
        <div className="flex items-start justify-between">
          <div>
            <h3 className="text-sm font-bold text-white">{farm.farm_name}</h3>
            <p className="text-[10px] text-[#4a5568] mt-0.5">{farm.total_area} {farm.area_unit}</p>
          </div>
          {vi && <VIReportBadge cvi_mean={vi.cvi_mean} />}
        </div>
      </div>

      <div className="px-4 pb-4 pt-3 space-y-3">
        {/* Coordinates */}
        <div className="flex items-center gap-2 text-xs text-[#7a90a8]">
          <svg className="w-3.5 h-3.5 text-[#4a5568] shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
              d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
          {farm.latitude?.toFixed(4)}°N, {farm.longitude?.toFixed(4)}°E
        </div>

        {/* Crop */}
        {crop ? (
          <div className="flex items-center gap-2">
            <div className="w-7 h-7 rounded-lg bg-[#1A6B3C]/10 border border-green-100 flex items-center justify-center shrink-0">
              <svg className="w-3.5 h-3.5" fill="none" stroke="#1A6B3C" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8}
                  d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />
              </svg>
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-xs font-semibold text-white truncate">{crop.crop_name}</p>
              <div className="flex items-center gap-1.5 mt-0.5">
                <SeasonPill season={crop.season} />
                <span className="text-[10px] text-[#4a5568]">Sown {formatDate(crop.sowing_date)}</span>
              </div>
            </div>
          </div>
        ) : (
          <p className="text-xs text-[#4a5568] italic">No crop recorded</p>
        )}

        {/* VI report */}
        {vi ? (
          <div className="bg-[#1a2432] border border-[#ffffff1a] rounded-lg p-3 space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-[10px] font-bold uppercase tracking-wider text-[#4a5568]">
                Satellite Report
              </span>
              <span className="text-[10px] text-[#4a5568]">
                {formatDate(vi.period_start)} – {formatDate(vi.period_end)}
              </span>
            </div>
            <div className="grid grid-cols-3 gap-2 text-center">
              {[
                { v: vi.cvi_mean?.toFixed(2),          k: 'CVI Mean' },
                { v: vi.ndvi?.toFixed(2),               k: 'NDVI' },
                { v: (vi.confidence_score?.toFixed(0) ?? '—') + '%', k: 'Confidence' },
              ].map(({ v, k }) => (
                <div key={k}>
                  <p className="text-base font-bold text-white">{v ?? '—'}</p>
                  <p className="text-[10px] text-[#4a5568]">{k}</p>
                </div>
              ))}
            </div>
          </div>
        ) : (
          <div className="bg-[#1a2432] border border-[#ffffff1a] rounded-lg p-3 text-center">
            <p className="text-xs text-[#4a5568]">Satellite analysis pending</p>
          </div>
        )}
      </div>
    </div>
  );
}
