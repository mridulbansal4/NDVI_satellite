/**
 * Step4_MapSelection.jsx — Step 4 of 9
 * Full-screen Leaflet map with leaflet-draw polygon tool.
 * Centroid + GeoJSON polygon saved to OnboardingContext on confirm.
 * No manual lat/lng inputs — coordinates auto-extracted from drawn polygon centroid.
 */

import { useState, useRef, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { MapContainer, TileLayer, FeatureGroup, Polygon, useMap } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet-draw';
import 'leaflet/dist/leaflet.css';
import 'leaflet-draw/dist/leaflet.draw.css';
import * as turf from '@turf/turf';
import { useOnboarding } from '../../context/OnboardingContext';

// Fix Leaflet default icon
delete L.Icon.Default.prototype._getIconUrl;
L.Icon.Default.mergeOptions({
  iconRetinaUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.7.1/images/marker-icon-2x.png',
  iconUrl:       'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.7.1/images/marker-icon.png',
  shadowUrl:     'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.7.1/images/marker-shadow.png',
});

// ── Draw control — must live inside MapContainer ─────────────────────────────
function DrawControlEffect({ featureGroupRef, onPolygonDrawn }) {
  const map = useMap();

  useEffect(() => {
    if (!map || !featureGroupRef.current) return;

    const drawControl = new L.Control.Draw({
      position: 'topright',
      draw: {
        rectangle:    false,
        circle:       false,
        circlemarker: false,
        marker:       false,
        polyline:     false,
        polygon: {
          allowIntersection: false,
          drawError: { color: '#ef4444', message: 'Intersections not allowed.' },
          shapeOptions: {
            color:       '#1A6B3C',
            weight:      3,
            fillColor:   '#1A6B3C',
            fillOpacity: 0.18,
          },
        },
      },
      edit: { featureGroup: featureGroupRef.current },
    });

    map.addControl(drawControl);

    const onCreate = (e) => {
      // Clear any previous polygon from the feature group
      featureGroupRef.current.clearLayers();
      featureGroupRef.current.addLayer(e.layer);

      const geojson = e.layer.toGeoJSON();
      onPolygonDrawn(geojson.geometry);
    };

    map.on(L.Draw.Event.CREATED, onCreate);

    return () => {
      map.removeControl(drawControl);
      map.off(L.Draw.Event.CREATED, onCreate);
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [map]);

  return null;
}

// ── Polygon renderer for already-confirmed polygon ───────────────────────────
function DrawnPolygon({ geometry }) {
  if (!geometry?.coordinates?.[0]) return null;
  const positions = geometry.coordinates[0].map(([lng, lat]) => [lat, lng]);
  return (
    <Polygon
      positions={positions}
      pathOptions={{ color: '#1A6B3C', weight: 3, fillOpacity: 0.18, fillColor: '#1A6B3C' }}
    />
  );
}

// ── Main Step 4 component ────────────────────────────────────────────────────
export default function Step4_MapSelection() {
  const navigate = useNavigate();
  const { state, merge, advanceStep } = useOnboarding();
  const featureGroupRef = useRef(null);

  // Restore any previously drawn polygon from context
  const [drawnGeom, setDrawnGeom] = useState(state.boundary_geom || null);
  const [area, setArea]           = useState(state.boundary_area_ha || null);
  const [centroid, setCentroid]   = useState(
    state.latitude && state.longitude
      ? { lat: state.latitude, lng: state.longitude }
      : null
  );

  const handlePolygonDrawn = useCallback((geometry) => {
    const feature  = { type: 'Feature', geometry, properties: {} };
    const areaSqM  = turf.area(feature);
    const areaHa   = areaSqM / 10000;
    const cent     = turf.centroid(feature);

    setDrawnGeom(geometry);
    setArea(areaHa);
    setCentroid({
      lat: cent.geometry.coordinates[1],
      lng: cent.geometry.coordinates[0],
    });
  }, []);

  const handleRedraw = () => {
    featureGroupRef.current?.clearLayers();
    setDrawnGeom(null);
    setArea(null);
    setCentroid(null);
  };

  const handleConfirm = () => {
    merge({
      latitude:         centroid.lat,
      longitude:        centroid.lng,
      boundary_geom:    drawnGeom,
      boundary_area_ha: area,
    });
    advanceStep(5, {
      latitude:      centroid.lat,
      longitude:     centroid.lng,
      boundary_geom: drawnGeom,
    });
    navigate('/onboarding/step5');
  };

  const areaAcres = area ? (area * 2.47105).toFixed(2) : null;

  return (
    <div
      className="fixed inset-0 flex flex-col"
      style={{ fontFamily: 'Inter, sans-serif', zIndex: 40 }}
    >
      {/* ── Top bar ─────────────────────────────────────────────────────── */}
      <div
        className="absolute top-0 left-0 right-0 bg-[#111820] border-b border-[#ffffff1a] flex items-center gap-3 px-4 py-3"
        style={{ zIndex: 1000 }}
      >
        <button
          onClick={() => navigate('/onboarding/step3')}
          className="flex items-center gap-1 text-sm text-[#7a90a8] hover:text-white transition-colors"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
          Back
        </button>

        <div className="flex-1">
          <p className="text-[10px] font-semibold uppercase tracking-widest text-[#1A6B3C]">Step 4 of 9</p>
          <h1 className="text-sm font-bold text-white leading-tight">Draw Your Farm Boundary</h1>
        </div>

        {/* Step dots */}
        <div className="flex gap-1">
          {Array.from({ length: 9 }).map((_, i) => (
            <div
              key={i}
              className={`rounded-full ${i < 4 ? 'bg-[#1A6B3C]' : 'bg-gray-200'}`}
              style={{ width: 8, height: 8 }}
            />
          ))}
        </div>
      </div>

      {/* ── Leaflet map ──────────────────────────────────────────────────── */}
      <div className="flex-1" style={{ marginTop: 61 }}>
        <MapContainer
          center={[20.5937, 78.9629]}
          zoom={14}
          style={{ width: '100%', height: '100%' }}
          zoomControl={false}
        >
          <TileLayer
            url="http://{s}.google.com/vt/lyrs=y&x={x}&y={y}&z={z}"
            subdomains={['mt0', 'mt1', 'mt2', 'mt3']}
            maxZoom={20}
            attribution="Map data &copy; Google"
          />

          <FeatureGroup ref={featureGroupRef}>
            <DrawControlEffect
              featureGroupRef={featureGroupRef}
              onPolygonDrawn={handlePolygonDrawn}
            />
          </FeatureGroup>

          {/* Show confirmed polygon from context on re-enter */}
          {drawnGeom && !featureGroupRef.current?.getLayers().length && (
            <DrawnPolygon geometry={drawnGeom} />
          )}
        </MapContainer>
      </div>

      {/* ── Instruction toast — shown when no polygon yet ────────────────── */}
      {!drawnGeom && (
        <div
          className="absolute bottom-6 left-1/2 bg-[#111820] border border-[#ffffff1a] rounded-xl shadow-lg px-5 py-3 text-center"
          style={{ zIndex: 1000, transform: 'translateX(-50%)', maxWidth: 300 }}
        >
          <p className="text-xs font-semibold text-white">
            Use the polygon tool (top-right corner) to draw your farm boundary
          </p>
          <p className="text-[10px] text-[#4a5568] mt-1">Click to add points · Double-click to finish</p>
        </div>
      )}

      {/* ── Confirm panel — shown after polygon is drawn ──────────────────── */}
      {drawnGeom && centroid && (
        <div
          className="absolute bottom-0 left-0 right-0 bg-[#111820] border-t border-[#ffffff1a] shadow-2xl px-4 pt-4 pb-6"
          style={{ zIndex: 1000 }}
        >
          <div className="max-w-md mx-auto space-y-3">
            {/* Calculated summary */}
            <div className="flex items-center justify-between bg-[#1a2432] rounded-xl px-4 py-3">
              <div>
                <p className="text-[10px] text-[#7a90a8] uppercase tracking-wider font-semibold">Calculated Area</p>
                <p className="text-base font-bold text-white mt-0.5">
                  {areaAcres} acres
                  <span className="text-sm font-normal text-[#4a5568] ml-2">({area?.toFixed(2)} ha)</span>
                </p>
              </div>
              <div className="text-right">
                <p className="text-[10px] text-[#7a90a8] uppercase tracking-wider font-semibold">Centroid</p>
                <p className="text-xs font-mono text-[#e2e8f0] mt-0.5">
                  {centroid.lat.toFixed(5)}°N<br />{centroid.lng.toFixed(5)}°E
                </p>
              </div>
            </div>

            {/* Action buttons */}
            <div className="flex gap-3">
              <button
                onClick={handleRedraw}
                className="flex-1 h-12 rounded-xl border border-[#ffffff1a] text-sm font-semibold text-[#e2e8f0] bg-[#111820] hover:bg-[#1a2432] transition-colors"
              >
                Redraw
              </button>
              <button
                onClick={handleConfirm}
                className="flex-[2] h-12 rounded-xl text-white text-sm font-bold transition-colors flex items-center justify-center gap-2"
                style={{ backgroundColor: '#1A6B3C' }}
                onMouseEnter={(e) => (e.currentTarget.style.backgroundColor = '#155c33')}
                onMouseLeave={(e) => (e.currentTarget.style.backgroundColor = '#1A6B3C')}
              >
                Confirm Location
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
