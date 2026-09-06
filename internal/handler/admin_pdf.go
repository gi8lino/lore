package handler

import (
	"cmp"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

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

// TestAdminPDFService renders and returns Lore's fixed two-page PDF diagnostic document.
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

		result, cleanup, err := pdf.RenderTest(r.Context(), pdfURL)
		if err != nil {
			logger.Warn("PDF service test failed", "event", "pdf_service_test_failed", "error", err)
			httpresponse.Problem(
				w,
				http.StatusBadGateway,
				"PDF service test failed.",
				httpresponse.NewFieldProblem("pdf_url", err.Error()),
			)
			return
		}
		defer cleanup()

		const filename = "lore-pdf-service-test.pdf"

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename=%q`, filename))
		w.Header().Set("X-Lore-PDF-Pages", strconv.Itoa(result.PageCount))
		w.Header().Set("X-Lore-PDF-Size", strconv.FormatInt(result.Size, 10))
		http.ServeContent(w, r, filename, time.Now(), result.File)
	}
}

// effectivePDFURL returns the deployment override when configured, otherwise the persisted endpoint.
func effectivePDFURL(runtimeOverride, persisted string) string {
	return cmp.Or(strings.TrimSpace(runtimeOverride), strings.TrimSpace(persisted))
}
