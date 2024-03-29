//  Copyright (c) 2017-2018 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"github.com/uber/aresdb/api/common"
	queryCom "github.com/uber/aresdb/query/common"
	"github.com/uber/aresdb/query/sql"
	"github.com/uber/aresdb/utils"
	"net/http"
)

// HandleSQL swagger:route POST /query/sql querySQL
// query in SQL
//
// Consumes:
//    - application/json
//    - application/hll
//
// Produces:
//    - application/json
//
// Responses:
//    default: errorResponse
//        200: aqlResponse
//        400: aqlResponse
func (handler *QueryHandler) HandleSQL(w http.ResponseWriter, r *http.Request) {
	sqlRequest := common.SQLRequest{Device: -1}

	if err := common.ReadRequest(r, &sqlRequest); err != nil {
		common.RespondWithBadRequest(w, err)
		utils.GetLogger().With(
			"error", err,
			"statusCode", http.StatusBadRequest,
		).Error("failed to parse query")
		return
	}

	var aqlQueries []queryCom.AQLQuery
	if sqlRequest.Body.Queries != nil {
		aqlQueries = make([]queryCom.AQLQuery, len(sqlRequest.Body.Queries))
		startTs := utils.Now()
		for i, sqlQuery := range sqlRequest.Body.Queries {
			parsedAQLQuery, err := sql.Parse(sqlQuery, utils.GetLogger())
			if err != nil {
				common.RespondWithBadRequest(w, err)
				return
			}
			aqlQueries[i] = *parsedAQLQuery
		}
		sqlParseTimer := utils.GetRootReporter().GetTimer(utils.QuerySQLParsingLatency)
		duration := utils.Now().Sub(startTs)
		sqlParseTimer.Record(duration)

	}

	aqlRequest := common.AQLRequest{
		Device:                sqlRequest.Device,
		Verbose:               sqlRequest.Verbose + sqlRequest.Debug,
		Debug:                 sqlRequest.Debug,
		Profiling:             sqlRequest.Profiling,
		DeviceChoosingTimeout: sqlRequest.DeviceChoosingTimeout,
		Accept:                sqlRequest.Accept,
		Origin:                sqlRequest.Origin,
		Body: queryCom.AQLRequest{
			Queries: aqlQueries,
		},
	}
	handler.handleAQLInternal(aqlRequest, w, r)
}
