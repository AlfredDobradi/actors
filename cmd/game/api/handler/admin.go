package handler

import (
	"log/slog"
	"net/http"

	"github.com/alfreddobradi/actors/cmd/game/api/state"
	"github.com/alfreddobradi/actors/cmd/game/api/template"
	"github.com/alfreddobradi/actors/cmd/game/model"
	"github.com/alfreddobradi/actors/cmd/game/repository"
	"github.com/alfreddobradi/actors/pkg/telemetry"
	"github.com/gorilla/mux"
)

func HandleAdminGetAccounts(s *state.Context) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		span := telemetry.SpanFromRequest(r)
		accounts, err := repository.GetAccountsKV(span.Context(), s.KV)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		data := struct {
			Accounts []model.Account
		}{
			Accounts: accounts,
		}
		w.Header().Set("content-type", "text/html")
		tpl, err := template.Render("account_list", data)
		if err != nil {
			slog.Error("failed to render template", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if _, err := w.Write(tpl); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
}

func HandleAdminGetAccount(s *state.Context) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		span := telemetry.SpanFromRequest(r)

		vars := mux.Vars(r)
		accountID := vars["accountId"]

		account, err := repository.GetAccountKV(span.Context(), s.KV, accountID)
		if err != nil {
			slog.Error("failed to fetch account", "error", err.Error(), "account_id", account.ID)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		sessions, err := repository.GetSessionsByAccountIDKV(span.Context(), s.KV, account.ID.String())
		if err != nil {
			slog.Error("failed to fetch sessions", "error", err.Error(), "account_id", account.ID)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		data := struct {
			Account  model.Account
			Sessions []model.Session
		}{
			Account:  account,
			Sessions: sessions,
		}

		w.Header().Set("content-type", "text/html")
		tpl, err := template.Render("account", data)
		if err != nil {
			slog.Error("failed to render template", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if _, err := w.Write(tpl); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
}
