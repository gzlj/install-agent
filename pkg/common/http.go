package common

type Response struct {
	Code int `json:"code"`
	Message string `json:"message"`
	Data interface{} `json:"data"`
}

func BuildResponse(code int, message string, data interface{}) (response Response ){
	response = Response{
		Code: code,
		Message: message,
		Data: data,
	}
	return
}
