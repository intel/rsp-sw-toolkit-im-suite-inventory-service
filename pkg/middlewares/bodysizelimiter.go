/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package middlewares

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/web"
)

// max size limit of body 16MB
const (
	requestMaxSize = 16 << 20
)

// Bodylimiter middleware
func Bodylimiter(next web.Handler) web.Handler {
	return web.Handler(func(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {
		if request.Method == http.MethodPost || request.Method == http.MethodPut {
			tracerID := ctx.Value(web.KeyValues).(*web.ContextValues).TraceID

			//check based on content length
			headerSet := request.Header.Get("Content-Length")
			if headerSet != "" && request.ContentLength > requestMaxSize {
				log.WithFields(log.Fields{
					"Method":     request.Method,
					"RequestURI": request.RequestURI,
					"TraceID":    tracerID,
					"Code":       http.StatusRequestEntityTooLarge,
				}).Error("Request entity too large")
				return web.ErrEntityTooLarge
			}

			// If header not set, set content length based on actual size of the body
			if headerSet == "" {
				var buf bytes.Buffer
				reqBody := http.MaxBytesReader(writer, request.Body, requestMaxSize)
				bodySize, err := buf.ReadFrom(reqBody)
				if err != nil {
					log.WithFields(log.Fields{
						"Method":     request.Method,
						"RequestURI": request.RequestURI,
						"TraceID":    tracerID,
						"Code":       http.StatusRequestEntityTooLarge,
					}).Error("Request entity too large")
					return web.ErrEntityTooLarge
				}
				request.Header.Set("Content-Length", strconv.Itoa(int(bodySize)))
				request.Body = ioutil.NopCloser(&buf)
			}
		}
		err := next(ctx, writer, request)
		return err
	})

}
