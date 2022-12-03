package rest

import (
	"github.com/fasthttp/router"
)

type Route interface {
	Config() RouteConfig
	Handler(ctx *Ctx) APIError
}

type Router = router.Router

type RouteConfig struct {
	URI        string
	Method     RouteMethod
	Children   []Route
	Middleware []Middleware
}

type RouteMethod string

const (
	GET     RouteMethod = "GET"
	POST    RouteMethod = "POST"
	PUT     RouteMethod = "PUT"
	PATCH   RouteMethod = "PATCH"
	DELETE  RouteMethod = "DELETE"
	OPTIONS RouteMethod = "OPTIONS"
)

type Middleware = func(ctx *Ctx) APIError

type APIErrorResponse struct {
	StatusCode HttpStatusCode         `json:"status_code"`
	Status     string                 `json:"status"`
	Error      string                 `json:"error"`
	ErrorCode  int                    `json:"error_code"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

type HttpStatusCode int

const (
	// 1xx Informational
	Continue          HttpStatusCode = 100
	SwitchingProtocol HttpStatusCode = 101
	Processing        HttpStatusCode = 102
	EarlyHints        HttpStatusCode = 103

	// 2xx Successful
	OK                          HttpStatusCode = 200
	Created                     HttpStatusCode = 201
	Accepted                    HttpStatusCode = 202
	NonAuthoritativeInformation HttpStatusCode = 203
	NoContent                   HttpStatusCode = 204
	ResetContent                HttpStatusCode = 205
	PartialContent              HttpStatusCode = 206
	MultiStatus                 HttpStatusCode = 207
	AlreadyReported             HttpStatusCode = 208
	IMUsed                      HttpStatusCode = 226

	// 3xx Redirections
	MultipleChoice    HttpStatusCode = 300
	MovedPermanently  HttpStatusCode = 301
	Found             HttpStatusCode = 302
	SeeOther          HttpStatusCode = 303
	NotModified       HttpStatusCode = 304
	TemporaryRedirect HttpStatusCode = 307
	PermanentRedirect HttpStatusCode = 308

	// 4xx Client Errors
	BadRequest                  HttpStatusCode = 400
	Unauthorized                HttpStatusCode = 401
	PaymentRequired             HttpStatusCode = 402
	Forbidden                   HttpStatusCode = 403
	NotFound                    HttpStatusCode = 404
	MethodNotAllowed            HttpStatusCode = 405
	NotAcceptable               HttpStatusCode = 406
	ProxyAuthenticationRequired HttpStatusCode = 407
	RequestTimeout              HttpStatusCode = 408
	Conflict                    HttpStatusCode = 409
	Gone                        HttpStatusCode = 410
	LengthRequired              HttpStatusCode = 411
	PreconditionFailed          HttpStatusCode = 412
	PayloadTooLarge             HttpStatusCode = 413
	URITooLong                  HttpStatusCode = 414
	UnsupportedMediaType        HttpStatusCode = 415
	RangeNotSatisfiable         HttpStatusCode = 416
	ExpectationFailed           HttpStatusCode = 417
	ImATeapot                   HttpStatusCode = 418
	MisdirectedRequest          HttpStatusCode = 421
	UnprocessableEntity         HttpStatusCode = 422
	Locked                      HttpStatusCode = 423
	FailedDependency            HttpStatusCode = 424
	TooEarly                    HttpStatusCode = 425
	UpgradeRequired             HttpStatusCode = 426
	PreconditionRequired        HttpStatusCode = 428
	TooManyRequests             HttpStatusCode = 429
	RequestHeaderFieldsTooLarge HttpStatusCode = 431
	UnavailableForLegalReasons  HttpStatusCode = 451

	// 5xx Server Errors
	InternalServerError           HttpStatusCode = 500
	NotImplemented                HttpStatusCode = 501
	BadGateway                    HttpStatusCode = 502
	ServiceUnavailable            HttpStatusCode = 503
	GatewayTimeout                HttpStatusCode = 504
	HttpVersionNotSupported       HttpStatusCode = 505
	VariantAlsoNegotiates         HttpStatusCode = 506
	InsufficientStorage           HttpStatusCode = 507
	LoopDetected                  HttpStatusCode = 508
	NotExtended                   HttpStatusCode = 510
	NetworkAuthenticationRequired HttpStatusCode = 511
)

// String: return the http status code in text form
func (c HttpStatusCode) String() string {
	return codeTextMap[c]
}

var codeTextMap = map[HttpStatusCode]string{
	Continue:                      "Continue",
	SwitchingProtocol:             "Switching Protocol",
	Processing:                    "Processing",
	EarlyHints:                    "Early Hints",
	OK:                            "OK",
	Created:                       "Created",
	Accepted:                      "Accepted",
	NonAuthoritativeInformation:   "Non-Authoritative Iformation",
	NoContent:                     "No Content",
	ResetContent:                  "Reset Content",
	PartialContent:                "Partial Content",
	MultiStatus:                   "Multi-Status",
	AlreadyReported:               "Already Reported",
	IMUsed:                        "IM Used",
	MultipleChoice:                "Multiple Choice",
	MovedPermanently:              "Moved Permanently",
	Found:                         "Found",
	SeeOther:                      "See Other",
	NotModified:                   "Not Modified",
	TemporaryRedirect:             "Temporary Redirect",
	PermanentRedirect:             "Permanent Redirect",
	BadRequest:                    "Bad Request",
	Unauthorized:                  "Unauthorized",
	PaymentRequired:               "Payment Required",
	Forbidden:                     "Forbidden",
	NotFound:                      "Not Found",
	MethodNotAllowed:              "Method Not Allowed",
	NotAcceptable:                 "Not Acceptable",
	ProxyAuthenticationRequired:   "Proxy Authentication Required",
	RequestTimeout:                "Request Timeout",
	Conflict:                      "Conflict",
	Gone:                          "Gone",
	LengthRequired:                "Length Required",
	PreconditionFailed:            "Precondition Failed",
	PayloadTooLarge:               "Payload Too Large",
	URITooLong:                    "URI Too Long",
	UnsupportedMediaType:          "Unsupported Media Type",
	RangeNotSatisfiable:           "Range Not Satisfiable",
	ExpectationFailed:             "Expectation Failed",
	ImATeapot:                     "I'm a teapot",
	MisdirectedRequest:            "Misredirected Request",
	UnprocessableEntity:           "Unprocessable Entity",
	Locked:                        "Locked",
	FailedDependency:              "Failed Dependency",
	TooEarly:                      "Too Early",
	UpgradeRequired:               "Upgrade Required",
	PreconditionRequired:          "Precondition Required",
	TooManyRequests:               "Too Many Requests",
	RequestHeaderFieldsTooLarge:   "Request Header Fields Too Large",
	UnavailableForLegalReasons:    "Unavailable For Legal Reasons",
	InternalServerError:           "Internal Server Error",
	NotImplemented:                "Not Implemented",
	BadGateway:                    "Bad Gateway",
	ServiceUnavailable:            "Service Unavailable",
	GatewayTimeout:                "Gateway Timeout",
	HttpVersionNotSupported:       "HTTP Version Not Supported",
	VariantAlsoNegotiates:         "Variant Also Negotiates",
	InsufficientStorage:           "Insufficient Storage",
	LoopDetected:                  "Loop Detected",
	NotExtended:                   "Not Extended",
	NetworkAuthenticationRequired: "Network Authentication Required",
}

type Map map[string]interface{}
