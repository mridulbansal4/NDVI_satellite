package firestore

import (
	"context"
	"math"

	fs "cloud.google.com/go/firestore"
)

// Collections managed here (§7.10):
//
//	/farmer_sessions/{farmer_id}  — onboarding resume state
//	/farm_alerts/{farm_id}        — VI dashboard alerts
//
// Every function is BEST-EFFORT: a Firestore failure is logged at WARN and
// never fails the request. That is what makes the onboarding flow work in dev,
// where Firestore is not configured at all.

// WriteSession records the farmer's onboarding progress so a low-connectivity
// client can resume. merge:true so a later step does not clobber earlier
// partial_data.
func (c *Client) WriteSession(ctx context.Context, farmerID string, step int, partial map[string]any) {
	db, err := c.DB(ctx)
	if err != nil {
		c.log.Warn("[Firestore] Session write failed (non-fatal): " + err.Error())
		return
	}
	_, err = db.Collection("farmer_sessions").Doc(farmerID).Set(ctx, map[string]any{
		"current_step": step,
		"partial_data": partial,
		"last_active":  fs.ServerTimestamp,
	}, fs.MergeAll)
	if err != nil {
		c.log.Warn("[Firestore] Session write failed (non-fatal): " + err.Error())
		return
	}
	c.log.Info("[Firestore] Session written for farmer " + farmerID)
}

// DeleteSession removes the onboarding session once consent is submitted,
// signalling that the farmer's data is committed to PostgreSQL.
func (c *Client) DeleteSession(ctx context.Context, farmerID string) {
	db, err := c.DB(ctx)
	if err != nil {
		c.log.Warn("[Firestore] Session delete failed (non-fatal): " + err.Error())
		return
	}
	if _, err := db.Collection("farmer_sessions").Doc(farmerID).Delete(ctx); err != nil {
		c.log.Warn("[Firestore] Session delete failed (non-fatal): " + err.Error())
		return
	}
	c.log.Info("[Firestore] Session deleted for farmer " + farmerID + " (onboarding complete)")
}

// VIData is the input to WriteFarmAlert.
type VIData struct {
	CVIMean float64
	NDVI    float64
	NDMI    float64
}

// WriteFarmAlert writes the dashboard alert document.
//
// The derivation is ported exactly from firestore/session.py (§7.10):
//
//	crop_health  = cvi_mean >= 0.6 ? Good : cvi_mean >= 0.4 ? Moderate : Poor
//	water_stress = ndmi >= 0.3 ? Low : ndmi >= 0.1 ? Moderate : High
//
// Today this is only ever called with zeros from the consent step's mock
// trigger, which is preserved as a mock (§7.10).
func (c *Client) WriteFarmAlert(ctx context.Context, farmID string, vi VIData) {
	cropHealth := "Poor"
	switch {
	case vi.CVIMean >= 0.6:
		cropHealth = "Good"
	case vi.CVIMean >= 0.4:
		cropHealth = "Moderate"
	}

	waterStress := "High"
	switch {
	case vi.NDMI >= 0.3:
		waterStress = "Low"
	case vi.NDMI >= 0.1:
		waterStress = "Moderate"
	}

	db, err := c.DB(ctx)
	if err != nil {
		c.log.Warn("[Firestore] Farm alert write failed (non-fatal): " + err.Error())
		return
	}
	_, err = db.Collection("farm_alerts").Doc(farmID).Set(ctx, map[string]any{
		"crop_health":      cropHealth,
		"vegetation_index": math.Round(vi.NDVI*1e4) / 1e4,
		"water_stress":     waterStress,
		"weather_alert":    "None",
		"pest_risk":        "Low",
		"last_updated":     fs.ServerTimestamp,
	})
	if err != nil {
		c.log.Warn("[Firestore] Farm alert write failed (non-fatal): " + err.Error())
		return
	}
	c.log.Info("[Firestore] Farm alert written for farm " + farmID + " (health=" + cropHealth + ")")
}
