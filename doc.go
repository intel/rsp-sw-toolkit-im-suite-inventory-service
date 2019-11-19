// Inventory Service.
//
// Inventory Microservice.
//
//     Schemes: http
//     Version: 1.0.0
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//
// swagger:meta
package main

// Forbidden
//swagger:response forbidden
type forbidden struct {
}

// External Error
//swagger:response externalError
type externalError struct {
}

// Service Unavailable
//swagger:response serviceUnavailable
type serviceUnavailable struct {
}

// External Service Timeout
//swagger:response externalServiceTimeout
type externalServiceTimeout struct {
}

// Internal Error
//swagger:response internalError
type internalError struct {
}
