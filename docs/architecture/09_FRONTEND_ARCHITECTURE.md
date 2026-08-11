# 09 — Frontend Architecture

## Component Structure (`frontend/src/`)

```
src/
├── main.jsx                   # Application entry point
├── App.jsx                    # Root component; holds active field & multi-field array
├── api.js                     # Centralized Axios client targeting Flask backend
├── firebase.js                # Firebase Web SDK initialization (Phone Auth)
├── colorUtils.js              # Legend palette color generator
├── pages/                     # Full-page routes
│   ├── Welcome.jsx            # Landing page
│   ├── Login.jsx              # Mobile OTP login
│   ├── Signup.jsx             # Onboarding registration
│   ├── Analysis.jsx           # Primary map & agronomy dashboard
│   └── steps/                 # 9-step onboarding forms
├── components/                # Modular UI components
│   ├── MapView.jsx            # Leaflet map container & draw controls
│   ├── HeatmapLayer.jsx       # GeoJSON grid renderer
│   ├── FarmSummary.jsx        # Index statistics sidebar & charts
│   ├── TimelineBar.jsx        # S2 scene acquisition date picker
│   ├── LayerToggle.jsx        # Optical vs Radar mode toggle
│   ├── Legend.jsx             # Dynamic index color scale legend
│   ├── KrishiMitraPanel.jsx   # Chatbot conversational drawer
│   ├── LocationForm.jsx       # India Post PIN code resolution form
│   └── PremiumAuthFlow.jsx    # Premium OTP login modal controller
└── context/                   # React Context
    └── OnboardingContext.jsx  # Multi-step state provider
```

## State Architecture
- **Multi-Field Storage (`App.jsx`)**: Maintains `fields[]` state array allowing users to switch between drawn polygons without re-querying backend.
- **Context API (`OnboardingContext.jsx`)**: Manages 9-step registration state across onboarding screens.

## Related Documents
- [13_MAP_RENDERING.md](./13_MAP_RENDERING.md)
- [14_LAYER_SYSTEM.md](./14_LAYER_SYSTEM.md)
- [22_COMPONENTS.md](../dashboard/22_COMPONENTS.md)
