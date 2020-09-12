package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HandleFunc func(ctx context.Context, req Request) (string, error)

type Request struct {
	JobID string `json:"id,omitempty"`
	Data  Data   `json:"data,omitempty"`
}

// [address:mm dataPrefix:qq functionSelector:aa result:0x00000000000000000000000000000000000000000000000000000000000fc646]
type Data struct {
	ContractAddress string `json:"address,omitempty"`
	DataPrefix      string `json:"dataPrefix,omitempty"`
	Function        string `json:"functionSelector,omitempty"`
	Result          string `json:"result,omitempty"`
}

type Response struct {
	JobRunID   string `json:"jobRunID,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
	Status     string `json:"status,omitempty"`
	Data       Data   `json:"data,omitempty"`
	Error      string `json:"error,omitempty"`
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

		res, err := handler(c.Request.Context(), req)
		if err != nil {
			log.Println("Handler error: ", err)
			errorResponse(c, http.StatusInternalServerError, req.JobID, "")
			return
		}

		c.JSON(http.StatusOK, Response{
			JobRunID:   req.JobID,
			StatusCode: http.StatusOK,
			Status:     "success",
			Data:       Data{Result: res},
		})
	})

	r.POST("/test", func(c *gin.Context) {
		var req Request
		if err := c.BindJSON(&req); err != nil {
			errorResponse(c, http.StatusBadRequest, req.JobID, "Invalid JSON payload")
			return
		}
		if err := validateRequest(&req); err != nil {
			errorResponse(c, http.StatusBadRequest, req.JobID, err.Error())
			return
		}
		log.Printf("incoming request: %+v", req)
		c.JSON(http.StatusOK, Response{
			JobRunID:   req.JobID,
			StatusCode: http.StatusOK,
			Status:     "success",
			Data:       Data{Result: "success"},
		})
	})
	return r
}
