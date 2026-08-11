package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS mirrors the app-wide flask_cors configuration in app.py.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §9.3.
//
// Applied globally, to every route group including /chatbot and /auth. This is
// not optional: vite.config.js proxies only /api, so the axios client reaches
// /auth, /farmer, /farm and /dashboard at http://localhost:5000 directly and
// genuinely needs CORS headers on them (see K9).
func CORS(origins []string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true, // supports_credentials=True
		MaxAge:           12 * time.Hour,
	})
}
