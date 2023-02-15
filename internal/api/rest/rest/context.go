package rest

import (
	"encoding/json"

	"github.com/seventv/api/internal/constant"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

type Ctx struct {
	*fasthttp.RequestCtx
}

type APIError = errors.APIError

func (c *Ctx) JSON(status HttpStatusCode, v interface{}) APIError {
	b, err := json.Marshal(v)
	if err != nil {
		c.SetStatusCode(InternalServerError)

		return errors.ErrInternalServerError().
			SetDetail("JSON Parsing Failed").
			SetFields(errors.Fields{"JSON_ERROR": err.Error()})
	}

	c.SetStatusCode(status)
	c.SetContentType("application/json")
	c.SetBody(b)

	return nil
}

func (c *Ctx) SetStatusCode(code HttpStatusCode) {
	c.RequestCtx.SetStatusCode(int(code))
}

func (c *Ctx) StatusCode() HttpStatusCode {
	return HttpStatusCode(c.RequestCtx.Response.StatusCode())
}

// Set the current authenticated user
func (c *Ctx) SetActor(u structures.User) {
	c.SetUserValue(string(constant.UserKey), u)
}

// Get the current authenticated user
func (c *Ctx) GetActor() (structures.User, bool) {
	v := c.RequestCtx.UserValue(constant.UserKey)
	switch v := v.(type) {
	case structures.User:
		return v, true
	default:
		return structures.DeletedUser, false
	}
}

func (c *Ctx) Log() *zap.SugaredLogger {
	z := zap.S().Named("api/rest").With(
		"request_id", c.ID(),
		"route", c.Path(),
	)

	actor, ok := c.GetActor()
	if ok {
		z = z.With("actor_id", actor.ID)
	}

	return z
}
