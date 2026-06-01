package service

import (
	"fmt"

	"github.com/gokins/gokins/comm"
	"github.com/sirupsen/logrus"
)

// GetIdOrAid looks up a record by id first, then falls back to aid.
// Returns true if the record was found.
// DB errors are logged but do not change the return value (false = not found or error).
func GetIdOrAid(id interface{}, e interface{}) bool {
	if id == nil || e == nil {
		return false
	}
	if v, ok := id.(string); ok && v == "" {
		return false
	}
	ok, err := comm.Db.Where("id=?", id).Get(e)
	if err != nil {
		logrus.Errorf("GetIdOrAid(id=%v) query error: %v", id, err)
		return false
	}
	if !ok {
		ok, err = comm.Db.Where("aid=?", id).Get(e)
		if err != nil {
			logrus.Errorf("GetIdOrAid(aid=%v) fallback query error: %v", id, err)
			return false
		}
	}
	return ok
}

// GetIdOrAidE is like GetIdOrAid but returns errors instead of logging them.
// Use this in contexts where callers need to distinguish "not found" from "DB error".
func GetIdOrAidE(id interface{}, e interface{}) (bool, error) {
	if id == nil || e == nil {
		return false, nil
	}
	if v, ok := id.(string); ok && v == "" {
		return false, nil
	}
	ok, err := comm.Db.Where("id=?", id).Get(e)
	if err != nil {
		return false, fmt.Errorf("GetIdOrAidE(id=%v): %w", id, err)
	}
	if !ok {
		ok, err = comm.Db.Where("aid=?", id).Get(e)
		if err != nil {
			return false, fmt.Errorf("GetIdOrAidE(aid=%v): %w", id, err)
		}
	}
	return ok, nil
}
