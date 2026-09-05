package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gi8lino/lore/internal/httpresponse"
	"github.com/gi8lino/lore/internal/pdf"
)

// SaveAdminPDFSettings stores the database-managed PDF rendering endpoint.
func SaveAdminPDFSettings(settingsUseCases settingsService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin := currentUser(r)
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid PDF settings form.")
			return
		}

		pdfURL := strings.TrimSpace(r.FormValue("pdf_url"))
		if err := pdf.ValidateURL(pdfURL); err != nil {
			httpresponse.Problem(
				w,
				http.StatusUnprocessableEntity,
				"PDF settings validation failed.",
				httpresponse.NewFieldProblem("pdf_url", err.Error()),
			)
			return
		}
		if err := settingsUseCases.SavePDFSettings(r.Context(), pdfURL, admin.ID); err != nil {
			writeAdminProblem(logger, w, err, "PDF settings")
			return
		}

		http.Redirect(w, r, "/admin/configuration#pdf-rendering", http.StatusSeeOther)
	}
}

// TestAdminPDFService verifies that a candidate endpoint can render a small PDF document.
func TestAdminPDFService(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid PDF service test request.")
			return
		}

		pdfURL := strings.TrimSpace(r.FormValue("pdf_url"))
		if pdfURL == "" {
			httpresponse.Problem(
				w,
				http.StatusUnprocessableEntity,
				"PDF service test failed.",
				httpresponse.NewFieldProblem("pdf_url", "Enter a PDF service URL to test."),
			)
			return
		}
		if err := pdf.ValidateURL(pdfURL); err != nil {
			httpresponse.Problem(
				w,
				http.StatusUnprocessableEntity,
				"PDF service test failed.",
				httpresponse.NewFieldProblem("pdf_url", err.Error()),
			)
			return
		}
		if err := pdf.Check(r.Context(), pdfURL); err != nil {
			logger.Warn("PDF service test failed", "event", "pdf_service_test_failed", "error", err)
			httpresponse.Problem(
				w,
				http.StatusBadGateway,
				"PDF service test failed.",
				httpresponse.NewFieldProblem("pdf_url", err.Error()),
			)
			return
		}

		httpresponse.Respond(w, http.StatusOK, map[string]string{
			"message": "PDF service rendered a test document successfully.",
		})
	}
}

// effectivePDFURL returns the deployment override when configured, otherwise the persisted endpoint.
func effectivePDFURL(runtimeOverride, persisted string) string {
	if runtimeOverride = strings.TrimSpace(runtimeOverride); runtimeOverride != "" {
		return runtimeOverride
	}
	return strings.TrimSpace(persisted)
}
