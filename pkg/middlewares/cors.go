/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package middlewares

import (
	"context"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/pkg/web"
	"net/http"
)

// CORS middleware
func CORS(origin string, next web.Handler) web.Handler {
	return web.Handler(func(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {
		writer.Header().Set("Access-Control-Allow-Origin", origin)
		writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		writer.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding")

		err := next(ctx, writer, request)
		return err
	})
}
