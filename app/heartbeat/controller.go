/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package heartbeat

import (
	"database/sql"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/pkg/jsonrpc"
)

func ProcessHeartbeat(hb *jsonrpc.Heartbeat, masterDB *sql.DB) error {

	// todo: heartbeat does not contain this data anymore
	//// Default coefficients
	//var coefficients facility.Coefficients
	//coefficients.DailyInventoryPercentage = config.AppConfig.DailyInventoryPercentage
	//coefficients.ProbUnreadToRead = config.AppConfig.ProbUnreadToRead
	//coefficients.ProbInStoreRead = config.AppConfig.ProbInStoreRead
	//coefficients.ProbExitError = config.AppConfig.ProbExitError
	//
	//// Insert facilities to database and set default coefficients if new facility is inserted
	//if err := facility.Insert(copySession, &facilityData, coefficients); err != nil {
	//	copySession.Close()
	//	return errors.Wrap(err, "Error replacing facilities")
	//}
	//copySession.Close()

	return nil
}
