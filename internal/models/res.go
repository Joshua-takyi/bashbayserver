package models

type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Page    int         `json:"page,omitempty"`
	Limit   int         `json:"limit,omitempty"`
	Total   int         `json:"total,omitempty"`
}

func SuccessResponse(data interface{}, message string) ApiResponse {
	return ApiResponse{
		Success: true,
		Data:    data,
		Message: message,
	}
}

func ErrorResponse(err string) ApiResponse {
	return ApiResponse{
		Success: false,
		Error:   err,
	}
}

func PaginatedResponse(data interface{}, page, limit, total int) ApiResponse {
	return ApiResponse{
		Success: true,
		Data:    data,
		Page:    page,
		Limit:   limit,
		Total:   total,
	}
}
