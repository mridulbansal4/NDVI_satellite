import React, { useState } from 'react';

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

/**
 * Legend — Collapsible sidebar panel showing the index color legend.
 * Rendered inside the sidebar, above the FarmSummary.
 */
export default function Legend({ activeLayer }) {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className="map-legend-overlay">
      <section className="card sidebar-legend" id="card-legend">
        <button
          className="sidebar-legend__toggle"
          onClick={() => setIsOpen(!isOpen)}
          aria-expanded={isOpen}
          type="button"
        >
          <span className="sidebar-legend__title">Index Legend</span>
          <span className={`sidebar-legend__chevron ${isOpen ? 'is-open' : ''}`}>▼</span>
        </button>

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
