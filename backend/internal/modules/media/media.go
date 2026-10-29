package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"log/slog"
	"net/http"
	"time"

	_ "image/gif" // decoders accepted on upload
	_ "image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	idc "github.com/NADER-CY/vira/backend/internal/modules/identity/contract"
	"github.com/NADER-CY/vira/backend/internal/modules/media/contract"
	socialc "github.com/NADER-CY/vira/backend/internal/modules/social/contract"
	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
	"github.com/NADER-CY/vira/backend/internal/platform/auth"
	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/httpx"
	"github.com/NADER-CY/vira/backend/internal/platform/ids"
	"github.com/NADER-CY/vira/backend/internal/platform/objectstore"
	"github.com/NADER-CY/vira/backend/internal/platform/ratelimit"
)

// Upload limits (P2 basic images; the full pipeline with variants and video is P3, ADR-009).
const (
	MaxUploadBytes = 10 << 20
	MaxPixels      = 40_000_000 // decompression-bomb guard
	MaxEdge        = 2048       // stored images are scaled down to fit
	JPEGQuality    = 85
)

// RuleUploads limits image uploads per user.
var RuleUploads = ratelimit.Rule{Name: "media_uploads", Limit: 60, Window: time.Hour}

// Service stores and serves images.
type Service struct {
	Pool    *db.Pool
	Store   objectstore.Store
	Limiter *ratelimit.Limiter
	Log     *slog.Logger
}

// Process decodes an uploaded image (JPEG, PNG, GIF, WebP), bounds its size and re-encodes it as JPEG. Re-encoding
// drops all metadata (EXIF GPS location, camera serials) — doc 08 sensitive-data protection.
func Process(data []byte) ([]byte, int, int, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, apperr.Validation(apperr.FieldError{Field: "file", Code: "unsupported_image"})
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > MaxPixels {
		return nil, 0, 0, apperr.Validation(apperr.FieldError{Field: "file", Code: "too_large"})
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, apperr.Validation(apperr.FieldError{Field: "file", Code: "unsupported_image"})
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > MaxEdge || h > MaxEdge {
		if w >= h {
			h, w = h*MaxEdge/w, MaxEdge
		} else {
			w, h = w*MaxEdge/h, MaxEdge
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	// White background for images with transparency (JPEG has no alpha).
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: JPEGQuality}); err != nil {
		return nil, 0, 0, err
	}
	return out.Bytes(), w, h, nil
}

// Uploaded is the API representation of a stored image.
type Uploaded struct {
	contract.Info
	Purpose string `json:"purpose"`
}

// Upload stores an image for owner.
func (s *Service) Upload(ctx context.Context, owner, purpose string, data []byte) (Uploaded, error) {
	if purpose != contract.PurposePost && purpose != contract.PurposeAvatar {
		return Uploaded{}, apperr.Validation(apperr.FieldError{Field: "purpose", Code: "invalid"})
	}
	if s.Limiter != nil {
		if err := s.Limiter.Check(ctx, false, ratelimit.Keyed{Rule: RuleUploads, Key: owner}); err != nil {
			return Uploaded{}, err
		}
	}
	out, w, h, err := Process(data)
	if err != nil {
		return Uploaded{}, err
	}
	id := ids.New()
	key := purpose + "/" + id[:2] + "/" + id + ".jpg"
	if err := s.Store.Put(ctx, key, "image/jpeg", out); err != nil {
		return Uploaded{}, apperr.Wrap(apperr.Unavailable, err)
	}
	sum := sha256.Sum256(out)
	if _, err := s.Pool.Exec(ctx, `INSERT INTO media.assets (id, owner_id, purpose, content_type, bytes, width, height, sha256, object_key)
		VALUES ($1, $2, $3, 'image/jpeg', $4, $5, $6, $7, $8)`, id, owner, purpose, len(out), w, h, sum[:], key); err != nil {
		return Uploaded{}, err
	}
	return Uploaded{Info: contract.Info{ID: id, URL: contract.URL(id), Width: w, Height: h}, Purpose: purpose}, nil
}

// Routes registers the media endpoints (signed-in only).
func (s *Service) Routes(r *httpx.Router, session httpx.Middleware) {
	r.HandleFunc(http.MethodPost, "/v1/media/images", s.hUpload, session)
	r.HandleFunc(http.MethodGet, "/v1/media/{id}", s.hContent, session)
}

func (s *Service) hUpload(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	if !idc.CanWrite(r.Context()) {
		httpx.WriteError(w, r, apperr.New(apperr.AccountRestricted, ""))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadBytes+1<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		httpx.WriteError(w, r, apperr.Validation(apperr.FieldError{Field: "file", Code: "too_large"}))
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()
	f, _, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, r, apperr.Validation(apperr.FieldError{Field: "file", Code: "required"}))
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxUploadBytes+1))
	if err != nil || len(data) > MaxUploadBytes {
		httpx.WriteError(w, r, apperr.Validation(apperr.FieldError{Field: "file", Code: "too_large"}))
		return
	}
	out, err := s.Upload(r.Context(), p.Subject, r.FormValue("purpose"), data)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, out)
}

func (s *Service) hContent(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	id := r.PathValue("id")
	if !ids.Valid(id) {
		httpx.WriteError(w, r, apperr.New(apperr.NotFound, ""))
		return
	}
	var owner, key string
	err := s.Pool.QueryRow(r.Context(), `SELECT owner_id, object_key FROM media.assets WHERE id = $1 AND state = 'ready'`, id).Scan(&owner, &key)
	if err != nil {
		httpx.WriteError(w, r, apperr.New(apperr.NotFound, ""))
		return
	}
	if owner != p.Subject {
		ok, err := socialc.MediaVisible(r.Context(), s.Pool, p.Subject, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if !ok {
			httpx.WriteError(w, r, apperr.New(apperr.NotFound, ""))
			return
		}
	}
	rc, ct, err := s.Store.Get(r.Context(), key)
	if errors.Is(err, objectstore.ErrNotFound) {
		httpx.WriteError(w, r, apperr.New(apperr.NotFound, ""))
		return
	}
	if err != nil {
		httpx.WriteError(w, r, apperr.Wrap(apperr.Unavailable, err))
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "private, max-age=86400, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = io.Copy(w, rc)
}
