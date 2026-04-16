import React from 'react';

export default function FarmSummary({ analysisData, activeField }) {
    if (!analysisData) return null;

    const farmArgs = analysisData.farm_summary;
    if (!farmArgs) return null;

    const indices = [
        { name: 'NDVI', label: 'Primary vegetation' },
        { name: 'EVI', label: 'Canopy density' },
        { name: 'SAVI', label: 'Soil-adjusted' },
        { name: 'NDMI', label: 'Moisture' },
        { name: 'GNDVI', label: 'Chlorophyll' },
    ];

    return (
        <section className="card card--results" id="card-results" aria-labelledby="results-heading">
            <h2 className="card__title" id="results-heading">
                Farm summary
            </h2>

            <div className="summary-grid" id="summary-grid" role="region" aria-label="Vegetation summary metrics">
                <div className="summary-row summary-row--split">
                    <div className="summary-block">
                        <span className="summary-cell__label">Field</span>
                        <span className="summary-cell__value summary-cell__value--compact">
                            {activeField?.name || 'Selected field'}
                        </span>
                    </div>
                    <div className="summary-block summary-block--end">
                        <span className="summary-cell__label">Area</span>
                        <span className="summary-cell__value summary-cell__value--compact summary-cell__value--accent">
                            {activeField?.areaHectares || 0} ha
                        </span>
                    </div>
                </div>

                <div className="summary-row">
                    <span className="summary-cell__label">Composite vegetation index (CVI)</span>
                    <span className="summary-cell__value summary-cell__value--compact">
                        {farmArgs.indices?.CVI?.mean?.toFixed(4) || 'N/A'}
                    </span>
                    <span className="summary-cell__sub">{farmArgs.indices?.CVI?.interpretation || ''}</span>
                </div>

                <div className="summary-row summary-row--split">
                    <div className="summary-block">
                        <span className="summary-cell__label">Confidence</span>
                        <span className="summary-cell__value summary-cell__value--compact">
                            {(farmArgs.confidence * 100).toFixed(1)}%
                        </span>
                    </div>
                    <div className="summary-block summary-block--end">
                        <span className="summary-cell__label">Clean scenes</span>
                        <span className="summary-cell__value summary-cell__value--compact">
                            {farmArgs.scene_count || 0}
                        </span>
                    </div>
                </div>

                <div className="summary-metrics" role="list">
                    {indices.map((b) => (
                        <div key={b.name} className="summary-metric" role="listitem">
                            <span className="summary-metric__name">{b.name}</span>
                            <span className="summary-metric__value">
                                {farmArgs.indices?.[b.name]?.mean?.toFixed(4) || 'N/A'}
                            </span>
                            <span className="summary-metric__hint">{b.label}</span>
                        </div>
                    ))}
                </div>
            </div>

            <button
                className="btn btn--secondary btn--sm"
                id="btn-clear"
                type="button"
                onClick={() => window.location.reload()}
            >
                Clear session
            </button>
        </section>
    );
}
