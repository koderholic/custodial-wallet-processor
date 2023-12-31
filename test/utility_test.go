package test

import (
	"testing"
	"time"
	"wallet-adapter/utility"

	"github.com/magiconair/properties/assert"
)

func TestCachePurgesAfterSetTime(t *testing.T) {

	expiry, purgeInterval := 5*time.Second, 10*time.Second
	newCache := utility.InitializeCache(expiry, purgeInterval)

	testKey, testValue := "test", "boy"

	newCache.Set(testKey, testValue, true)
	itemFetch := newCache.Get(testKey)
	if testValue != itemFetch {
		t.Errorf("Expected item fetched to be %s, got %s\n", testValue, itemFetch)
	}

	time.Sleep(purgeInterval)

	itemFetch = newCache.Get("test")
	if nil != itemFetch {
		t.Errorf("Expected item fetched to be empty %s, got %s\n", "<nil>", itemFetch)
	}
}

func TestCacheNeverExpires(t *testing.T) {

	expiry, purgeInterval := 5*time.Second, 10*time.Second
	newCache := utility.InitializeCache(expiry, purgeInterval)

	testKey, testValue := "test", "boy"

	newCache.Set(testKey, testValue, false)
	itemFetch := newCache.Get(testKey)
	if testValue != itemFetch {
		t.Errorf("Expected item fetched to be %s, got %s\n", testValue, itemFetch)
	}

	time.Sleep(purgeInterval)

	itemFetch = newCache.Get("test")
	if nil == itemFetch {
		t.Errorf("Expected item fetched to be %s, got %s\n", testValue, itemFetch)
	}

}

func TestCacheSetAndGetsProperly(t *testing.T) {

	type TestSetCache struct {
		Testdata string
	}

	expiry, purgeInterval := 5*time.Second, 10*time.Second
	newCache := utility.InitializeCache(expiry, purgeInterval)

	testKey1, testValue1 := "test", "boy"

	newCache.Set(testKey1, testValue1, true)
	itemFetch := newCache.Get(testKey1)
	if testValue1 != itemFetch {
		t.Errorf("Expected item fetched to be %s, got %s\n", testValue1, itemFetch)
	}

	testKey2, testValue2 := "test", TestSetCache{
		Testdata: testValue1,
	}
	newCache.Set(testKey2, &testValue2, true)

	itemFetch2 := newCache.Get(testKey2).(*TestSetCache)
	if nil == itemFetch2 {
		t.Errorf("Expected item fetched to be %+v, got %+v\n", testValue2, itemFetch)
	}

}

func TestDecimalsOperations(t *testing.T) {
	assert.Equal(t, utility.Subtract(0.301123778899876, "3.1155667011223", 18), "2.814442922222424")
	assert.Equal(t, utility.Subtract(0.004, "0.0100415", 8), "0.0060415")

	assert.Equal(t, utility.Add(0.301123778899876, "3.1155667011223", 18), "3.416690480022176")
	assert.Equal(t, utility.Add(0.004, "0.0100415", 18), "0.0140415")

}

func TestIsExceedWaitTime(t *testing.T) {
	assert.Equal(t, utility.IsExceedWaitTime(time.Now().Add(24*time.Hour), time.Now()), true)
	assert.Equal(t, utility.IsExceedWaitTime(time.Now(), time.Now().Add(24*time.Hour)), false)
}


func TestGetNextDayFromNow(t *testing.T) {
	assert.Equal(t, utility.GetNextDayFromNow().Day(), time.Now().AddDate(0,0,1).Day())
}