package ssdb

import (
	"errors"
	"github.com/siddontang/golib/hack"
	"strconv"
)

var errEmptyKVKey = errors.New("invalid empty kv key")
var errKVKey = errors.New("invalid encode kv key")

func encode_kv_key(key []byte) []byte {
	ek := make([]byte, key+1)
	ek[0] = KV_TYPE
	copy(ek[1:], key)
	return ek
}

func decode_kv_key(encodeKey []byte) ([]byte, error) {
	if encodeKey[0] != KV_TYPE {
		return nil, errKVKey
	}

	return encodeKey[1:], nil
}

func (a *App) kv_get(key []byte) ([]byte, error) {
	key = encode_kv_key(key)

	return a.db.Get(key)
}

func (a *App) kv_set(key []byte, value []byte) error {
	key = encode_kv_key(key)
	var err error

	t := a.newTx()

	a.kvMutex.Lock()
	defer a.kvMutex.Unlock()

	t.Put(key, value)

	//todo, binlog

	err = t.Commit()

	return err
}

func (a *App) kv_getset(key []byte, value []byte) ([]byte, error) {
	key = encode_kv_key(key)
	var err error

	a.kvMutex.Lock()
	defer a.kvMutex.Unlock()

	oldValue, _ := a.db.Get(key)

	t := a.newTx()

	t.Put(key, value)
	//todo, binlog

	err = t.Commit()

	return oldValue, err
}

func (a *App) kv_setnx(key []byte, value []byte) (int64, error) {
	key = encode_kv_key(key)
	var err error

	var n int64 = 1

	t := a.newTx()

	a.kvMutex.Lock()
	defer a.kvMutex.Unlock()

	if v, _ := a.db.Get(key); v != nil {
		n = 0
	} else {
		t.Put(key, value)

		//todo binlog

		err = t.Commit()
	}

	return n, err
}

func (a *App) kv_exists(key []byte) (int64, error) {
	key = encode_kv_key(key)
	var err error

	var v []byte
	v, err = a.db.Get(key)
	if v != nil && err != nil {
		return 1, nil
	} else {
		return 0, err
	}
}

func (a *App) kv_incr(key []byte, delta int64) (int64, error) {
	key = encode_kv_key(key)
	var err error

	t := a.newTx()

	a.kvMutex.Lock()
	defer a.kvMutex.Unlock()

	var v []byte
	v, err = a.db.Get(key)
	if err != nil {
		return 0, err
	}

	var n int64 = 0
	if v != nil {
		n, err = strconv.ParseInt(hack.String(v), 10, 64)
		if err != nil {
			return 0, err
		}
	}

	n += delta

	t.Put(key, hack.Slice(strconv.FormatInt(n, 10)))

	//todo binlog

	err = t.Commit()
	return n, err
}

func (a *App) tx_del(keys [][]byte) (int64, error) {
	for i := range keys {
		keys[i] = encode_kv_key(keys[i])
	}

	t := a.newTx()

	a.kvMutex.Lock()
	defer a.kvMutex.Unlock()

	for i := range keys {
		t.Delete(keys[i])
		//todo binlog
	}

	err := t.Commit()
	return int64(len(keys)), err
}

func (a *App) tx_mset(args [][]byte) error {
	t := a.newTx()

	a.kvMutex.Lock()
	defer a.kvMutex.Unlock()

	for i := 0; i < len(args); i += 2 {
		key := encode_kv_key(args[i])
		value := args[i+1]

		t.Put(key, value)

		//todo binlog
	}

	err := t.Commit()
	return err
}

func (a *App) kv_mget(args [][]byte) ([]interface{}, error) {
	values := make([]interface{}, len(args))

	for i := range args {
		key := encode_kv_key(args[i])
		value, err := a.db.Get(key)
		if err != nil {
			return nil, err
		}

		values[i] = value
	}

	return values, nil
}
