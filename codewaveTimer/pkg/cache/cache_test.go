package cache

import (
	"bytes"
	"fmt"
	"math/rand"
	"path"
	"reflect"
	"runtime"
	"testing"
	"time"
)

var (
	//lock    = sync.Mutex{}
	randStr = rand.New(rand.NewSource(time.Now().Unix()))
	letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

func GetTestKey(i int) []byte {
	return []byte(fmt.Sprintf("easycache-test-key-%09d", i))
}

func RandomValue(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[randStr.Intn(len(letters))]
	}
	return []byte("easycache-test-value-" + string(b))
}

func noError(t *testing.T, e error) {
	if e != nil {
		_, file, line, _ := runtime.Caller(1)
		file = path.Base(file)
		t.Errorf(fmt.Sprintf("\n%s:%d: Error is not nil: \n"+
			"actual  : %T(%#v)\n", file, line, e, e))
	}
}

func assertEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	if !objectsAreEqual(expected, actual) {
		_, file, line, _ := runtime.Caller(1)
		file = path.Base(file)
		t.Errorf(fmt.Sprintf("\n%s:%d: Not equal: \n"+
			"expected: %T(%#v)\n"+
			"actual  : %T(%#v)\n",
			file, line, expected, expected, actual, actual), msgAndArgs...)
	}
}

func objectsAreEqual(expected, actual interface{}) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	exp, ok := expected.([]byte)
	if !ok {
		return reflect.DeepEqual(expected, actual)
	}

	act, ok := actual.([]byte)
	if !ok {
		return false
	}
	if exp == nil || act == nil {
		return exp == nil && act == nil
	}
	return bytes.Equal(exp, act)
}

func TestSetAndGet(t *testing.T) {
	t.Parallel()

	// given
	cache, _ := New(DefaultConfig())
	value := "value"

	// when
	cache.Set("key", value, 0*time.Second)
	cachedValue, err := cache.Get("key")

	// then
	noError(t, err)
	assertEqual(t, value, cachedValue)
}

func TestNotExist(t *testing.T) {
	t.Parallel()

	cache, _ := New(DefaultConfig())

	_, err := cache.Get("key")
	assertEqual(t, ErrKeyNotExist, err)
}

func TestSetAndDelayGet(t *testing.T) {
	t.Parallel()

	cache, _ := New(TestConfig())

	key := "easy_key"
	value := RandomValue(20)
	err := cache.Set(key, value, 3*time.Second)
	noError(t, err)

	cacheValue, _ := cache.Get(key)
	assertEqual(t, value, cacheValue)

	time.Sleep(4 * time.Second)

	_, err = cache.Get(key)
	assertEqual(t, ErrKeyNotExist, err)
}

func TestDelete(t *testing.T) {

	t.Parallel()
	cache, _ := New(TestConfig())

	key := string(RandomValue(20))
	cache.Set(key, 0, 0*time.Second)
	err := cache.Delete(key) // del persist key
	assertEqual(t, nil, err)

	key1 := string(RandomValue(20))
	cache.Set(key1, 0, 2*time.Second)
	time.Sleep(3 * time.Second)
	err = cache.Delete(key1) // del expire key
	assertEqual(t, ErrKeyNotExist, err)
}

func TestRemoveCallback(t *testing.T) {

	t.Parallel()

	conf := TestConfig()
	conf.OnRemoveWithReason = func(key string, value interface{}, reason RemoveReason) {
		t.Logf("key <%s> del reason is <%d>\n", key, reason)
	}
	cache, _ := New(conf)
	key := string(RandomValue(20))
	cache.Set(key, 0, 2*time.Second)
	key1 := string(RandomValue(20))
	cache.Set(key1, 0, 0*time.Second)
	cache.Delete(key1)

	time.Sleep(3 * time.Second)

}

func TestGetIfNotExist(t *testing.T) {
	t.Parallel()
	cache, _ := New(TestConfig())

	cache.Set("key", 1, 3*time.Second)

	time.Sleep(4 * time.Second)

	cacheValue, _ := cache.GetIfNotExist("key", GetterFunc(func(s string) (interface{}, error) {
		return "yay!this is soruce", nil
	}), 2*time.Second)

	assertEqual(t, "yay!this is soruce", cacheValue)
}
