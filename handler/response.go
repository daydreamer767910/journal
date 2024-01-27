package handler

type Meta interface{}

type jsonHTTPResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Meta    Meta   `json:"meta"`
}
