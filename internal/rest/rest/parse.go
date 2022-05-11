package rest

import (
	"strconv"

	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/Common/structures/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Param struct {
	v interface{}
}

func (c *Ctx) UserValue(key Key) *Param {
	return &Param{c.RequestCtx.UserValue(string(key))}
}

// String returns a string value of the param
func (p *Param) String() (string, bool) {
	if p.v == nil {
		return "", false
	}
	var s string
	switch t := p.v.(type) {
	case string:
		s = t
	default:
		return "", false
	}

	return s, true
}

// Int32 parses the param into an int32
func (p *Param) Int32() (int32, error) {
	s, ok := p.String()
	if !ok {
		return 0, errors.ErrEmptyField()
	}

	i, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, errors.ErrBadInt().SetDetail(err.Error())
	}
	return int32(i), nil
}

// Int64 parses the param into an int64
func (p *Param) Int64() (int64, error) {
	s, ok := p.String()
	if !ok {
		return 0, errors.ErrEmptyField()
	}

	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, errors.ErrBadInt().SetDetail(err.Error())
	}
	return int64(i), nil
}

// ObjectID parses the param into an Object ID
func (p *Param) ObjectID() (primitive.ObjectID, error) {
	s, _ := p.String()
	if s == "" || !primitive.IsValidObjectID(s) {
		return primitive.NilObjectID, errors.ErrBadObjectID()
	}

	oid, err := primitive.ObjectIDFromHex(s)
	if err != nil {
		return primitive.NilObjectID, errors.ErrBadObjectID().SetDetail(err.Error())
	}
	return oid, nil
}

func (p *Param) User() *structures.User {
	var u *structures.User
	switch t := p.v.(type) {
	case *structures.User:
		u = t
	default:
		return nil
	}
	return u
}
