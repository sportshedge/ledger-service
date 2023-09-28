package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"general_ledger_golang/models"
	"general_ledger_golang/pkg/app"
	"general_ledger_golang/pkg/e"
	"general_ledger_golang/pkg/logger"
	"general_ledger_golang/pkg/util"
	"general_ledger_golang/service/book_service"
)

func GetBook(c *gin.Context) {
	appGin := app.Gin{C: c}
	bookId := c.Param("bookId")
	balanceFetchStr := c.Query("balance")
	balanceFetch, err := strconv.ParseBool(balanceFetchStr)

	if err != nil {
		logger.Logger.Errorf("Parsing of `balance` failed, error: %+v", err)
	}

	bookService := book_service.BookService{}
	result, err := bookService.GetBook(bookId, balanceFetch)

	if err != nil {
		appGin.Response(http.StatusInternalServerError, e.ERROR, map[string]interface{}{"error": err.Error()})
		return
	}
	if result == nil {
		appGin.Response(http.StatusNotFound, e.NOT_EXIST, map[string]interface{}{"book": result})
		return
	}
	appGin.Response(http.StatusOK, e.SUCCESS, map[string]interface{}{"book": result})
	return
}

func GetBookBalance(c *gin.Context) {
	appGin := app.Gin{C: c}
	bookId := c.Param("bookId")

	assetId := c.Query("assetId")
	operationType := c.Query("operationType")

	bookService := book_service.BookService{}

	result, err := bookService.GetBalance(bookId, assetId, operationType, nil)

	if err != nil {
		appGin.Response(http.StatusInternalServerError, e.ERROR, map[string]interface{}{"error": err.Error()})
		return
	}

	appGin.Response(http.StatusOK, e.SUCCESS, result)
	return
}

func CreateOrUpdateBook(c *gin.Context) {
	appGin := app.Gin{C: c}
	reqBody := util.GetReqBodyFromCtx(c)

	if reqBody == nil {
		appGin.Response(http.StatusBadRequest, e.INVALID_PARAMS, map[string]interface{}{"error": "Missing request body or not a valid json!"})
		return
	}

	name := reqBody["name"].(string)
	metadataBytes, _ := json.Marshal(reqBody["metadata"])
	book := models.Book{Name: name, Metadata: datatypes.JSON(metadataBytes)}
	result, operation := book.CreateOrUpdateBook(&book)
	err := result.Error

	if err != nil {
		fmt.Printf("Book creation failed: %+v", err)
		appGin.Response(http.StatusInternalServerError, e.INVALID_PARAMS, map[string]interface{}{"error": err.Error()})
		return
	}

	appGin.Response(http.StatusOK, e.SUCCESS, map[string]interface{}{
		"book":    book,
		"message": fmt.Sprintf("%v successful", operation),
	})
	return
}
