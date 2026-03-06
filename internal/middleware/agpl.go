// Copyright 2026 kpp.dev
//
// This file is part of kpfc.
//
// kpfc is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// kpfc is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with kpfc. If not, see <https://www.gnu.org/licenses/>.

package middleware

import "net/http"

// AGPLSourceCode is a middleware that adds AGPL §13 compliance headers
// to all HTTP responses, providing users interacting with the software
// via network access to the source code location.
func AGPLSourceCode(sourceURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Source-Code", sourceURL)
			w.Header().Set("X-License", "AGPL-3.0-or-later")
			next.ServeHTTP(w, r)
		})
	}
}
