// Package handler implements HTTP handlers of the metrics collection API.
//
// Endpoints:
//
//	GET  /                              HTML list of all metrics
//	POST /update/{type}/{name}/{value}  update a single metric via URL
//	GET  /value/{type}/{name}           read a metric as plain text
//	POST /update                        update a metric via JSON
//	POST /value                         read a metric via JSON
//	POST /updates/                      update a batch of metrics via JSON
//	GET  /ping                          database health check
package handler
