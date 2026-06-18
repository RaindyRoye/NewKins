package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/sirupsen/logrus"
)

// FindParam looks up a param by name using the global context.
// Prefer FindParamCtx when a request context is available.
func FindParam(key string) (*model.TParam, bool) {
	return FindParamCtx(comm.Ctx, key)
}

// FindParamCtx looks up a param by name with the provided context for cancellation/timeout.
func FindParamCtx(ctx context.Context, key string) (*model.TParam, bool) {
	e := &model.TParam{}
	ok, err := comm.Db.Context(ctx).Where("name=?", key).Get(e)
	if err != nil {
		logrus.Errorf("FindParam(%s) err:%v", key, err)
	}
	return e, ok
}

// SetParam creates or updates a param using the global context.
// Prefer SetParamCtx when a request context is available.
func SetParam(key string, data []byte, tit ...string) error {
	return SetParamCtx(comm.Ctx, key, data, tit...)
}

// SetParamCtx creates or updates a param with the provided context.
func SetParamCtx(ctx context.Context, key string, data []byte, tit ...string) error {
	var err error
	db := comm.Db.Context(ctx)
	e, ok := FindParamCtx(ctx, key)
	if len(tit) > 0 {
		e.Title = tit[0]
	}
	e.Data = string(data)
	if ok && e.Aid > 0 {
		_, err = db.Cols("title", "data").Where("aid=?", e.Aid).Update(e)
		if err != nil {
			return fmt.Errorf("update param %q: %w", key, err)
		}
	} else {
		e.Name = key
		e.Times = time.Now()
		_, err = db.Insert(e)
		if err != nil {
			return fmt.Errorf("insert param %q: %w", key, err)
		}
	}
	return nil
}

// SetsParam serializes data as JSON and stores it using the global context.
// Prefer SetsParamCtx when a request context is available.
func SetsParam(key string, data any, tit ...string) error {
	return SetsParamCtx(comm.Ctx, key, data, tit...)
}

// SetsParamCtx serializes data as JSON and stores it with the provided context.
func SetsParamCtx(ctx context.Context, key string, data any, tit ...string) error {
	if data == nil {
		return ErrParamDataNil
	}
	bts, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal param data: %w", err)
	}
	return SetParamCtx(ctx, key, bts, tit...)
}

// GetParam retrieves raw param bytes using the global context.
// Prefer GetParamCtx when a request context is available.
func GetParam(key string) ([]byte, error) {
	return GetParamCtx(comm.Ctx, key)
}

// GetParamCtx retrieves raw param bytes with the provided context.
func GetParamCtx(ctx context.Context, key string) ([]byte, error) {
	e, ok := FindParamCtx(ctx, key)
	if ok {
		return []byte(e.Data), nil
	}
	return nil, ErrParamNotFound
}

// GetsParam deserializes a param into data using the global context.
// Prefer GetsParamCtx when a request context is available.
func GetsParam(key string, data any) error {
	return GetsParamCtx(comm.Ctx, key, data)
}

// GetsParamCtx deserializes a param into data with the provided context.
func GetsParamCtx(ctx context.Context, key string, data any) error {
	if data == nil {
		return ErrParamDataNil
	}
	bts, err := GetParamCtx(ctx, key)
	if err != nil {
		return fmt.Errorf("get param %q: %w", key, err)
	}
	if err := json.Unmarshal(bts, data); err != nil {
		return fmt.Errorf("unmarshal param %q data: %w", key, err)
	}
	return nil
}

// GetsParamCache retrieves a param with caching using the global context.
// Prefer GetsParamCacheCtx when a request context is available.
func GetsParamCache(key string, data any, outm ...time.Duration) error {
	return GetsParamCacheCtx(comm.Ctx, key, data, outm...)
}

// GetsParamCacheCtx retrieves a param with caching and the provided context.
func GetsParamCacheCtx(ctx context.Context, key string, data any, outm ...time.Duration) error {
	err := comm.CacheGets(key, data)
	if err == nil {
		return nil
	}
	err = GetsParamCtx(ctx, key, data)
	if err == nil {
		errs := comm.CacheSets(key, data, outm...)
		if errs != nil {
			logrus.Errorf("GetsParamCache.CacheSets(%s) err:%v", key, errs)
		}
	}
	return err
}
