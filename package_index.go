// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"net/http"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/Masterminds/semver/v3"
	"github.com/gorilla/mux"

	"github.com/elastic/package-registry/packages"
	"github.com/elastic/package-registry/util"
)

const (
	packageIndexRouterPath = "/package/{packageName:[a-z0-9_]+}/{packageVersion}/"
)

var errPackageRevisionNotFound = errors.New("package revision not found")

func packageIndexHandler(indexer Indexer, cacheTime time.Duration) func(w http.ResponseWriter, r *http.Request) {
	logger := util.Logger()
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		packageName, ok := vars["packageName"]
		if !ok {
			badRequest(w, "missing package name")
			return
		}

		packageVersion, ok := vars["packageVersion"]
		if !ok {
			badRequest(w, "missing package version")
			return
		}

		_, err := semver.StrictNewVersion(packageVersion)
		if err != nil {
			badRequest(w, "invalid package version")
			return
		}

		opts := packages.NameVersionFilter(packageName, packageVersion)
		packages, err := indexer.Get(r.Context(), &opts)
		if err != nil {
			logger.Error("getting package path failed", zap.Error(err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if len(packages) == 0 {
			notFoundError(w, errPackageRevisionNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		cacheHeaders(w, cacheTime)

		err = util.WriteJSONPretty(w, packages[0])
		if err != nil {
			logger.Error("marshaling package index failed",
				zap.String("package.path", packages[0].BasePath),
				zap.Error(err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}
}
