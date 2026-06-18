package comm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

var mainCacheBucket = []byte("mainCacheBucket")

// ErrCacheNotInit is returned when any cache operation is attempted
// before the cache has been initialized.
var ErrCacheNotInit = errors.New("cache not init")

// ErrCacheDataNil is returned when a nil data argument is passed to
// a cache setter that requires non-nil data.
var ErrCacheDataNil = errors.New("data must not be nil")

func CacheSet(key string, data []byte, outm ...time.Duration) error {
	if BCache == nil {
		return ErrCacheNotInit
	}
	err := BCache.Update(func(tx *bolt.Tx) error {
		var err error
		bk := tx.Bucket(mainCacheBucket)
		if bk == nil {
			bk, err = tx.CreateBucket(mainCacheBucket)
			if err != nil {
				return err
			}
		}
		if data == nil {
			return bk.Delete([]byte(key))
		}
		buf := &bytes.Buffer{}
		var outms []byte
		if len(outm) > 0 {
			outms = []byte(time.Now().Add(outm[0]).Format(time.RFC3339Nano))
		} else {
			outms = []byte(time.Now().Add(time.Hour).Format(time.RFC3339Nano))
		}
		buf.Write(hbtp.BigIntToByte(int64(len(outms)), 4))
		buf.Write(outms)
		buf.Write(data)
		return bk.Put([]byte(key), buf.Bytes())
	})
	if err != nil {
		return fmt.Errorf("cache set key %q: %w", key, err)
	}
	return nil
}
func CacheSets(key string, data any, outm ...time.Duration) error {
	if BCache == nil {
		return ErrCacheNotInit
	}
	if data == nil {
		return CacheSet(key, nil)
	}
	bts, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("cache marshal: %w", err)
	}
	return CacheSet(key, bts, outm...)
}
func parseCacheData(bts []byte) []byte {
	if len(bts) < 4 {
		return nil
	}
	ln := int(hbtp.BigByteToInt(bts[:4]))
	if len(bts) < 4+ln {
		return nil
	}
	tms := string(bts[4 : 4+ln])
	outm, err := time.Parse(time.RFC3339Nano, tms)
	if err != nil {
		return nil
	}
	if time.Since(outm).Milliseconds() < 0 {
		return bts[4+ln:]
	}
	return nil
}

var ErrKeyNotFound = errors.New("key not found")
var ErrKeyTimeout = errors.New("key is timeout")

func CacheGet(key string) ([]byte, error) {
	if BCache == nil {
		return nil, ErrCacheNotInit
	}
	var rt []byte
	var expired bool
	err := BCache.View(func(tx *bolt.Tx) error {
		bk := tx.Bucket(mainCacheBucket)
		if bk == nil {
			return ErrKeyNotFound
		}
		bts := bk.Get([]byte(key))
		if bts == nil {
			return ErrKeyNotFound
		}
		rt = parseCacheData(bts)
		if rt == nil {
			expired = true
			return ErrKeyTimeout
		}
		return nil
	})
	// Wrap unexpected bbolt errors while preserving sentinel errors for errors.Is checks.
	if err != nil && !errors.Is(err, ErrKeyNotFound) && !errors.Is(err, ErrKeyTimeout) {
		err = fmt.Errorf("cache get key %q: %w", key, err)
	}
	// Delete expired keys in a separate write transaction (View is read-only).
	if expired {
		go func() {
			if err := BCache.Update(func(tx *bolt.Tx) error {
				bk := tx.Bucket(mainCacheBucket)
				if bk == nil {
					return nil
				}
				return bk.Delete([]byte(key))
			}); err != nil {
				logrus.Warnf("CacheGet: failed to delete expired key %q: %v", key, err)
			}
		}()
	}
	if time.Since(mainCacheClearTime).Hours() > 30 {
		go mainCacheClear()
	}
	return rt, err
}
func CacheGets(key string, data any) error {
	if BCache == nil {
		return ErrCacheNotInit
	}
	if data == nil {
		return ErrCacheDataNil
	}
	bts, err := CacheGet(key)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bts, data); err != nil {
		return fmt.Errorf("cache unmarshal: %w", err)
	}
	return nil
}

func CacheFlush() error {
	if BCache == nil {
		return ErrCacheNotInit
	}
	err := BCache.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket(mainCacheBucket)
	})
	if err != nil {
		return fmt.Errorf("cache flush: %w", err)
	}
	return nil
}

var mainCacheClearTime time.Time

func mainCacheClear() {
	defer func() {
		if err := recover(); err != nil {
			logrus.Errorf("mainCacheClear recover err:%v", err)
		}
	}()

	if BCache == nil {
		return
	}
	/*if time.Now().Hour()!=3|| time.Since(mainCacheClearTime).Hours() < 30 {
		return
	}*/
	mainCacheClearTime = time.Now()
	/*if err := CacheFlush(); err != nil {
		logrus.Errorf("mainCacheClear err:%v", err)
	}*/
	var deleteErrors int
	err := BCache.Update(func(tx *bolt.Tx) error {
		bk := tx.Bucket(mainCacheBucket)
		if bk == nil {
			return nil
		}
		// Collect keys to delete first to avoid mutating during iteration.
		var toDelete [][]byte
		_ = bk.ForEach(func(k, v []byte) error {
			data := parseCacheData(v)
			if data == nil {
				toDelete = append(toDelete, append([]byte(nil), k...))
			}
			return nil
		})
		for _, k := range toDelete {
			if err := bk.Delete(k); err != nil {
				deleteErrors++
				logrus.Warnf("mainCacheClear: failed to delete expired key %q: %v", k, err)
			}
		}
		return nil
	})
	if err != nil {
		logrus.Errorf("mainCacheClear err:%v", err)
	}
	if deleteErrors > 0 {
		logrus.Warnf("mainCacheClear: %d keys could not be deleted", deleteErrors)
	}
}
