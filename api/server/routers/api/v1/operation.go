package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"general_ledger_golang/pkg/app"
	"general_ledger_golang/pkg/e"
	"general_ledger_golang/pkg/logger"
	"general_ledger_golang/pkg/util"
	"general_ledger_golang/service/operation_service"
)

func PostOperation(c *gin.Context) {
	appGin := app.Gin{C: c}
	reqBody := util.GetReqBodyFromCtx(c)

	opType := reqBody["type"]
	memo := reqBody["memo"]
	entries := reqBody["entries"]
	metadata := reqBody["metadata"]

	opMap := map[string]interface{}{
		"type":     opType,
		"memo":     memo,
		"entries":  entries,
		"metadata": metadata,
	}

	log := logger.Logger.WithFields(logrus.Fields{
		"memo": memo,
		"op":   opMap,
	})

	log.Infof("Request Received")

	opService := &operation_service.OperationService{}
	foundOp, err := opService.PostOperation(opMap)

	if err != nil || foundOp == nil {
		log.Errorf("Creating Operation Failed, error: %+v", err)
		appGin.Response(http.StatusInternalServerError, e.ERROR, map[string]interface{}{
			"message": "Creating operation resulted in error!",
			"error":   err.Error(),
		})
		return
	}

	// return the operation
	appGin.Response(http.StatusOK, e.SUCCESS, map[string]interface{}{"operation": foundOp})
	return
}

func GetOperationByMemo(c *gin.Context) {
	appGin := app.Gin{C: c}

	memo := c.Query("memo")

	if memo == "" {
		appGin.Response(http.StatusBadRequest, e.INVALID_PARAMS, map[string]interface{}{
			"message": "Memo is not provided!",
		})
		return
	}

	opService := &operation_service.OperationService{}
	foundOp, err := opService.GetOperation(memo, nil)

	if err != nil {
		logger.Logger.Errorf("Fetching Operation Failed, error: %+v", err)
		appGin.Response(http.StatusInternalServerError, e.ERROR, map[string]interface{}{
			"message": "Fetching operation resulted in error!",
		})
		return
	}

	status := e.SUCCESS
	httpStatus := http.StatusOK

	if foundOp == nil {
		status = e.NOT_EXIST
		httpStatus = http.StatusNotFound
	}
	// return the operation
	appGin.Response(httpStatus, status, map[string]interface{}{"operation": foundOp})
	return
}
