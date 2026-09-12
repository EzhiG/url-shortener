package handler

import (
	"net/http"
)

const cookieName = "token"

func setAuthCookie(auth AuthService, w http.ResponseWriter, userID string) (string, error) {
	if userID == "" {
		var err error
		userID, err = auth.GenerateUserID()
		if err != nil {
			return "", err
		}
	}

	token, err := auth.BuildToken(userID)
	if err != nil {
		return "", err
	}

	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: token, Path: "/", HttpOnly: true, MaxAge: 60 * 60 * 24 * 30})

	return userID, nil
}

func GetAuthCookie(r *http.Request) string {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}

	return cookie.Value
}

func NewAuthMiddleware(auth AuthService) func(next http.Handler) http.Handler {
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := GetAuthCookie(r)
			userID, err := auth.ParseToken(token)

			if err != nil {
				var newUserID string
				if auth.IsTokenExpiredOnlyError(err) {
					newUserID = userID
				} else {
					newUserID = ""
				}

				userID, err = setAuthCookie(auth, w, newUserID)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}

			newCtx := auth.WithUserID(r.Context(), userID)
			r = r.WithContext(newCtx)

			next.ServeHTTP(w, r)
		})
	}

	return mw
}
