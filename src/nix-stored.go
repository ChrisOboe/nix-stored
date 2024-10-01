//go:generate oapi-codegen --generate types,chi-server,strict-server --package api -o api/api.gen.go schemas/api.yaml

package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/ChrisOboe/nix-stored/api"
	"github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
)

type Authentication struct {
	User string
	Pass string
}

type Settings struct {
	StorePath       string
	ListenInterface string
	UserRead        Authentication
	UserWrite       Authentication
}

func defaultEnv(envVar string, def string) string {
	env := os.Getenv(envVar)
	if env != "" {
		return env
	}
	return def
}

func SettingsFromEnv() (Settings, error) {
	rpassfile := os.Getenv("NIX_STORED_USER_READ_PASSFILE")
	wpassfile := os.Getenv("NIX_STORED_USER_WRITE_PASSFILE")

	var ReadAuth Authentication
	ReadAuth.User = os.Getenv("NIX_STORED_USER_READ")

	var WriteAuth Authentication
	WriteAuth.User = os.Getenv("NIX_STORED_USER_WRITE")

	if rpassfile != "" {
		slog.Debug("Reading read user password file", "path", rpassfile)
		rpass, err := os.ReadFile(rpassfile)
		if err != nil {
			return Settings{}, fmt.Errorf("Couldn't read read passfile: %w", err)
		}
		ReadAuth.Pass = string(rpass)
	} else {
		ReadAuth.Pass = os.Getenv("NIX_STORED_USER_READ_PASS")
	}

	if wpassfile != "" {
		slog.Debug("Reading write user password file", "path", wpassfile)
		wpass, err := os.ReadFile(wpassfile)
		if err != nil {
			return Settings{}, fmt.Errorf("Couldn't read read passfile: %w", err)
		}
		WriteAuth.Pass = string(wpass)
	} else {
		WriteAuth.Pass = os.Getenv("NIX_STORED_USER_WRITE_PASS")
	}

	return Settings{
		StorePath:       defaultEnv("NIX_STORED_PATH", "/var/lib/nixStored"),
		ListenInterface: defaultEnv("NIX_STORED_LISTEN_INTERFACE", "127.0.0.1:8100"),
		UserRead:        ReadAuth,
		UserWrite:       WriteAuth,
	}, nil
}

func main() {
	consoleHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(consoleHandler))

	s, err := SettingsFromEnv()
	if err != nil {
		slog.Error("Couldn't load settings", "error", err)
		return
	}
	slog.Info("loaded settings", "settings", s)

	ns := NixStored{StorePath: s.StorePath}
	// create dirs
	err = os.MkdirAll(s.StorePath+"/nar", 0770)
	if err != nil {
		slog.Error("Couldn't create dir", "error", err)
		return
	}

	options := api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Warn("Request Error", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			_, e := w.Write([]byte(err.Error()))
			if e != nil {
				slog.Error("Couldn't write response", "error", err)
			}
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Error("Response Error", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			_, e := w.Write([]byte(err.Error()))
			if e != nil {
				slog.Error("Couldn't write response", "error", err)
			}
		},
	}

	apiHandler := api.NewStrictHandlerWithOptions(ns, []api.StrictMiddlewareFunc{BasicAuthMiddleware(s.UserRead, s.UserWrite), LogMiddleware()}, options)
	http.Handle("/", api.Handler(apiHandler))

	slog.Info("Starting http server", "interface", s.ListenInterface)
	err = http.ListenAndServe(s.ListenInterface, nil)
	if err != nil {
		slog.Error("Couldn't create webserver", "error", err)
		return
	}
}

type NixStored struct {
	StorePath string
}

// Get the build logs for a particular deriver. This path exists if this binary cache is hydrated from Hydra.
// (GET /log/{deriver})
func (n NixStored) GetDeriverBuildLog(ctx context.Context, request api.GetDeriverBuildLogRequestObject) (api.GetDeriverBuildLogResponseObject, error) {
	// not yet implemented
	return api.GetDeriverBuildLog501Response{}, nil
}

// Get the compressed NAR object
// (GET /nar/{fileHash}.nar.{compression})
func (n NixStored) GetCompressedNar(ctx context.Context, request api.GetCompressedNarRequestObject) (api.GetCompressedNarResponseObject, error) {
	filename := fmt.Sprintf("%s/nar/%s.nar.%s", n.StorePath, request.FileHash, request.Compression)
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return api.GetCompressedNar404Response{}, nil
		} else {
			slog.Error("Couldn't open file", "file", filename, "error", err)
			return api.GetCompressedNar500Response{}, nil
		}
	}
	info, err := file.Stat()
	if err != nil {
		slog.Error("Couldn't get fileinfo", "file", filename, "error", err)
		return api.GetCompressedNar500Response{}, nil
	}

	return api.GetCompressedNar200ApplicationxNixNarResponse{
		Body:          file,
		ContentLength: info.Size(),
	}, nil
}

// Check if the NAR is there
// (HEAD /nar/{fileHash}.nar.{compression})
func (n NixStored) HeadNarFileHashNarCompression(ctx context.Context, request api.HeadNarFileHashNarCompressionRequestObject) (api.HeadNarFileHashNarCompressionResponseObject, error) {
	filename := fmt.Sprintf("%s/nar/%s.nar.%s", n.StorePath, request.FileHash, request.Compression)
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return api.HeadNarFileHashNarCompression404Response{}, nil
		} else {
			slog.Error("Couldn't open file", "file", filename, "error", err)
			return api.HeadNarFileHashNarCompression500Response{}, nil
		}
	}
	return api.HeadNarFileHashNarCompression200Response{}, nil
}

// Upload NAR
// (PUT /nar/{fileHash}.nar.{compression})
func (n NixStored) PutNarFileHashNarCompression(ctx context.Context, request api.PutNarFileHashNarCompressionRequestObject) (api.PutNarFileHashNarCompressionResponseObject, error) {
	filename := fmt.Sprintf("%s/nar/%s.nar.%s", n.StorePath, request.FileHash, request.Compression)
	file, err := os.Create(filename)
	if err != nil {
		slog.Error("Couldn't open file", "file", filename, "error", err)
		return api.PutNarFileHashNarCompression500Response{}, nil
	}
	defer file.Close()
	_, err = io.Copy(file, request.Body)

	return api.PutNarFileHashNarCompression201Response{}, nil
}

// Get information about this Nix binary cache
// (GET /nix-cache-info)
func (n NixStored) GetNixCacheInfo(ctx context.Context, request api.GetNixCacheInfoRequestObject) (api.GetNixCacheInfoResponseObject, error) {
	return api.GetNixCacheInfo200JSONResponse{
		Priority:      30,
		StoreDir:      "/nix/store",
		WantMassQuery: 1,
	}, nil
}

// Get the file listings for a particular store-path (once you expand the NAR).
// (GET /{storePathHash}.ls)
func (n NixStored) GetNarFileListing(ctx context.Context, request api.GetNarFileListingRequestObject) (api.GetNarFileListingResponseObject, error) {
	slog.Warn("Get NarFileListing called")
	return api.GetNarFileListing501Response{}, nil
}

// Get the NarInfo for a particular path
// (GET /{storePathHash}.narinfo)
func (n NixStored) GetNarInfo(ctx context.Context, request api.GetNarInfoRequestObject) (api.GetNarInfoResponseObject, error) {
	filename := fmt.Sprintf("%s/%s.narinfo", n.StorePath, request.StorePathHash)

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return api.GetNarInfo404Response{}, nil
		} else {
			slog.Error("Couldn't open file", "file", filename, "error", err)
			return api.GetNarInfo500Response{}, nil
		}
	}
	info, err := file.Stat()
	if err != nil {
		slog.Error("Couldn't get fileinfo", "file", filename, "error", err)
		return api.GetNarInfo500Response{}, nil
	}

	return api.GetNarInfo200TextxNixNarinfoResponse{
		Body:          file,
		ContentLength: info.Size(),
	}, nil
}

// Check if a particular path exists quickly
// (HEAD /{storePathHash}.narinfo)
func (n NixStored) DoesNarInfoExist(ctx context.Context, request api.DoesNarInfoExistRequestObject) (api.DoesNarInfoExistResponseObject, error) {
	filename := fmt.Sprintf("%s/%s.narinfo", n.StorePath, request.StorePathHash)

	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return api.DoesNarInfoExist404Response{}, nil
		} else {
			slog.Error("Couldn't open file", "file", filename, "error", err)
			return api.DoesNarInfoExist500Response{}, nil
		}
	}
	return api.DoesNarInfoExist200Response{}, nil
}

// (PUT /{storePathHash}.narinfo)
func (n NixStored) PutStorePathHashNarinfo(ctx context.Context, request api.PutStorePathHashNarinfoRequestObject) (api.PutStorePathHashNarinfoResponseObject, error) {
	filename := fmt.Sprintf("%s/%s.narinfo", n.StorePath, request.StorePathHash)

	file, err := os.Create(filename)
	if err != nil {
		slog.Error("Couldn't open file", "file", filename, "error", err)
		return api.PutStorePathHashNarinfo500Response{}, nil
	}
	defer file.Close()
	_, err = io.Copy(file, request.Body)

	return api.PutStorePathHashNarinfo201Response{}, nil
}

func LogMiddleware() api.StrictMiddlewareFunc {
	return func(f nethttp.StrictHTTPHandlerFunc, operationID string) nethttp.StrictHTTPHandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (response interface{}, err error) {
			slog.Debug("REST API Called", "operation", operationID, "request", request)
			return f(ctx, w, r, request)
		}
	}
}

func BasicAuthMiddleware(ruser Authentication, rwUser Authentication) api.StrictMiddlewareFunc {
	return func(f nethttp.StrictHTTPHandlerFunc, operationID string) nethttp.StrictHTTPHandlerFunc {
		// nothing needs to be authenticated on auth none
		if ruser.User == "" && rwUser.User == "" {
			return f
		}

		switch operationID {
		case "PutNarFileHashNarCompression":
		case "PutStorePathHashNarinfo":
			return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (response interface{}, err error) {
				user, pass, ok := r.BasicAuth()
				if !ok {
					return nil, fmt.Errorf("Corrupt BasicAuth")
				}
				if user != rwUser.User || pass != rwUser.Pass {
					return nil, fmt.Errorf("Wrong Credentials")
				}
				return f(ctx, w, r, request)
			}
		}

		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (response interface{}, err error) {
			user, pass, ok := r.BasicAuth()
			if !ok {
				return nil, fmt.Errorf("Corrupt BasicAuth")
			}
			if (user == ruser.User && pass == ruser.Pass) || (user == rwUser.User && pass == rwUser.Pass) {
				return f(ctx, w, r, request)
			}
			return nil, fmt.Errorf("Wrong Credentials")
		}
	}
}
