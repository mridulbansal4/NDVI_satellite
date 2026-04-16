import React, { useState, useRef, useCallback, useEffect } from 'react';
import { GripVertical } from 'lucide-react';

const LEGEND_BINS = [
    { range: '0.95 – 1.00', label: 'Better to use NDRE', color: '#007e47' },
    { range: '0.90 – 0.95', label: 'Dense vegetation', color: '#009755' },
    { range: '0.85 – 0.90', label: 'Dense vegetation', color: '#14aa60' },
    { range: '0.80 – 0.85', label: 'Dense vegetation', color: '#53bd6b' },
    { range: '0.75 – 0.80', label: 'Dense vegetation', color: '#77ca6f' },
    { range: '0.70 – 0.75', label: 'Dense vegetation', color: '#9bd873' },
    { range: '0.65 – 0.70', label: 'Dense vegetation', color: '#b9e383' },
    { range: '0.60 – 0.65', label: 'Dense vegetation', color: '#d5ef94' },
    { range: '0.55 – 0.60', label: 'Moderate vegetation', color: '#eaf7ac' },
    { range: '0.50 – 0.55', label: 'Moderate vegetation', color: '#fdfec2' },
    { range: '0.45 – 0.50', label: 'Moderate vegetation', color: '#ffefab' },
    { range: '0.40 – 0.45', label: 'Moderate vegetation', color: '#ffe093' },
    { range: '0.35 – 0.40', label: 'Sparse vegetation', color: '#ffc67d' },
    { range: '0.30 – 0.35', label: 'Sparse vegetation', color: '#ffab69' },
    { range: '0.25 – 0.30', label: 'Sparse vegetation', color: '#ff8d5a' },
    { range: '0.20 – 0.25', label: 'Sparse vegetation', color: '#fe6c4a' },
    { range: '0.15 – 0.20', label: 'Open soil', color: '#ef4c3a' },
    { range: '0.10 – 0.15', label: 'Open soil', color: '#e02d2c' },
    { range: '0.05 – 0.10', label: 'Open soil', color: '#c5142a' },
    { range: '-1.00 – 0.05', label: 'Open soil', color: '#ad0028' },
];

const STORAGE_KEY = 'mx-legend-pos';

function readStoredPosition() {
    try {
        const raw = sessionStorage.getItem(STORAGE_KEY);
        if (!raw) return null;
        const p = JSON.parse(raw);
        if (typeof p.left === 'number' && typeof p.top === 'number') return p;
    } catch {}
    return null;
}

/**
 * Floating NDVI / index legend on the map — draggable so it stays clear of draw tools.
 */
export default function Legend({ activeLayer }) {
  const [isOpen, setIsOpen] = useState(false);
  const [pos, setPos] = useState(() => readStoredPosition() ?? { top: 16, left: 16 });
  const panelRef = useRef(null);
  const dragRef = useRef(null);

  const layerLabel = typeof activeLayer === 'string' ? activeLayer.toUpperCase() : 'Index';

  const clampToMap = useCallback((left, top) => {
    const wrap = panelRef.current?.closest('.map-wrapper');
    const el = panelRef.current;
    if (!wrap || !el) return { left, top };
    const m = 8;
    const w = wrap.clientWidth;
    const h = wrap.clientHeight;
    const pw = el.offsetWidth;
    const ph = el.offsetHeight;
    return {
      left: Math.max(m, Math.min(left, w - pw - m)),
      top: Math.max(m, Math.min(top, h - ph - m)),
    };
  }, []);

  useEffect(() => {
    const onResize = () => {
      setPos((p) => clampToMap(p.left, p.top));
    };
    window.addEventListener('resize', onResize);
    return () => window.removeEventListener('resize', onResize);
  }, [clampToMap]);

  const onHandlePointerDown = useCallback((e) => {
    if (e.button !== 0) return;
    e.preventDefault();
    const start = { x: e.clientX, y: e.clientY, left: pos.left, top: pos.top };
    dragRef.current = start;

    const onMove = (ev) => {
      const d = dragRef.current;
      if (!d) return;
      const dx = ev.clientX - d.x;
      const dy = ev.clientY - d.y;
      const next = clampToMap(d.left + dx, d.top + dy);
      setPos(next);
    };

    const onUp = () => {
      dragRef.current = null;
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup', onUp);
      window.removeEventListener('pointercancel', onUp);
      setPos((p) => {
        try {
          sessionStorage.setItem(STORAGE_KEY, JSON.stringify(p));
        } catch {}
        return p;
      });
    };

    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', onUp);
    window.addEventListener('pointercancel', onUp);
  }, [pos.left, pos.top, clampToMap]);

  return (
    <div
      ref={panelRef}
      className="map-legend-overlay"
      style={{
        top: pos.top,
        left: pos.left,
        right: 'auto',
      }}
    >
      <section className="card sidebar-legend" id="card-legend">
        <div className="sidebar-legend__header-row">
          <button
            type="button"
            className="sidebar-legend__handle"
            aria-label="Drag legend"
            title="Drag to move"
            onPointerDown={onHandlePointerDown}
          >
            <GripVertical size={14} strokeWidth={2} aria-hidden />
          </button>
          <button
            type="button"
            className="sidebar-legend__toggle"
            onClick={() => setIsOpen(!isOpen)}
            aria-expanded={isOpen}
          >
            <span className="sidebar-legend__title-wrap">
              <span className="sidebar-legend__title">Legend</span>
              <span className="sidebar-legend__subtitle">{layerLabel} scale</span>
            </span>
            <span className={`sidebar-legend__chevron ${isOpen ? 'is-open' : ''}`}>▼</span>
          </button>
        </div>

        {isOpen && (
          <div className="sidebar-legend__body">
            {LEGEND_BINS.map((bin, index) => (
              <div key={index} className="sidebar-legend__row">
                <div className="sidebar-legend__chip" style={{ backgroundColor: bin.color }} />
                <div className="sidebar-legend__range">{bin.range}</div>
                <div className="sidebar-legend__label">{bin.label}</div>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
