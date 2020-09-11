package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HandleFunc func(req Request) (string, error)

type Request struct {
	JobID string `json:"id"`
	Data  Data   `json:"data"`
}

// [address:mm dataPrefix:qq functionSelector:aa result:0x00000000000000000000000000000000000000000000000000000000000fc646]
type Data struct {
	ContractAddress string      `json:"address"`
	DataPrefix      string      `json:"dataPrefix"`
	Function        string      `json:"functionSelector"`
	Result          interface{} `json:"result"`
}

type Response struct {
	JobRunID   string `json:"jobRunID"`
	StatusCode int    `json:"status_code"`
	Status     string `json:"status"`
	Data       string `json:"data"`
	Error      string `json:"error"`
}

func validateRequest(r *Request) error {
	if r.JobID == "" || r.Data.Function == "" || r.Data.ContractAddress == "" {
		return errors.New("missing required field(s)")
	}
	return nil
}

func errorResponse(c *gin.Context, statusCode int, jobID, errMsg string) {
	if errMsg != "" {
		log.Println("Request error: ", errMsg)
	}
	c.JSON(statusCode, Response{
		JobRunID:   jobID,
		StatusCode: statusCode,
		Status:     "errored",
		Error:      errMsg,
	})
}

func NewServerRouter(handler HandleFunc) *gin.Engine {
	r := gin.Default()
	r.POST("/", func(c *gin.Context) {
		var req Request
		if err := c.BindJSON(&req); err != nil {
			errorResponse(c, http.StatusBadRequest, req.JobID, "Invalid JSON payload")
			return
		}
		if err := validateRequest(&req); err != nil {
			errorResponse(c, http.StatusBadRequest, req.JobID, err.Error())
			return
		}

		res, err := handler(req)
		if err != nil {
			log.Println("Handler error: ", err)
			errorResponse(c, http.StatusInternalServerError, req.JobID, "")
			return
		}

		c.JSON(http.StatusOK, Response{
			JobRunID:   req.JobID,
			StatusCode: http.StatusOK,
			Status:     "success",
			Data:       res,
		})
	})
	return r
}
