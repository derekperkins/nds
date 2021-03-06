package nds_test

import (
	"io"
	"reflect"
	"sync"
	"testing"

	"github.com/qedus/nds"

	"errors"

	"appengine"
	"appengine/aetest"
	"appengine/datastore"
	"appengine/memcache"
)

func TestGetMultiStruct(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Get from datastore.
	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if response[i].IntVal != i+1 {
			t.Fatal("incorrect IntVal")
		}
	}

	// Get from cache.
	response = make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if response[i].IntVal != i+1 {
			t.Fatal("incorrect IntVal")
		}
	}
}

func TestGetMultiStructPtr(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Get from datastore.
	response := make([]*testEntity, len(keys))
	for i := 0; i < len(response); i++ {
		response[i] = &testEntity{}
	}

	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if response[i].IntVal != i+1 {
			t.Fatal("incorrect IntVal")
		}
	}

	// Get from cache.
	response = make([]*testEntity, len(keys))
	for i := 0; i < len(response); i++ {
		response[i] = &testEntity{}
	}
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if response[i].IntVal != i+1 {
			t.Fatal("incorrect IntVal")
		}
	}
}

func TestGetMultiInterface(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Get from datastore.
	response := make([]interface{}, len(keys))
	for i := 0; i < len(response); i++ {
		response[i] = &testEntity{}
	}

	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if te, ok := response[i].(*testEntity); ok {
			if te.IntVal != i+1 {
				t.Fatal("incorrect IntVal")
			}
		} else {
			t.Fatal("incorrect type")
		}
	}

	// Get from cache.
	response = make([]interface{}, len(keys))
	for i := 0; i < len(response); i++ {
		response[i] = &testEntity{}
	}
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if te, ok := response[i].(*testEntity); ok {
			if te.IntVal != i+1 {
				t.Fatal("incorrect IntVal")
			}
		} else {
			t.Fatal("incorrect type")
		}
	}
}

func TestGetMultiPropertyLoadSaver(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int
	}

	keys := []*datastore.Key{}
	entities := []datastore.PropertyList{}

	for i := 1; i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", int64(i), nil))

		pl := datastore.PropertyList{}
		if err := nds.SaveStruct(&testEntity{i}, &pl); err != nil {
			t.Fatal(err)
		}
		entities = append(entities, pl)
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Prime the cache..
	uncachedEntities := make([]datastore.PropertyList, len(keys))
	if err := nds.GetMulti(c, keys, uncachedEntities); err != nil {
		t.Fatal(err)
	}

	for i, e := range entities {
		if !reflect.DeepEqual(e, uncachedEntities[i]) {
			t.Fatal("uncachedEntities not equal", e, uncachedEntities[i])
		}
	}

	// Use cache.
	cachedEntities := make([]datastore.PropertyList, len(keys))
	if err := nds.GetMulti(c, keys, cachedEntities); err != nil {
		t.Fatal(err)
	}

	for i, e := range entities {
		if !reflect.DeepEqual(e, cachedEntities[i]) {
			t.Fatal("cachedEntities not equal", e, cachedEntities[i])
		}
	}

	// We know the datastore supports property load saver but we need to make
	// sure that memcache does by ensuring memcache does not error when we
	// change to fetching with structs.
	// Do this by making sure the datastore is not called on this following
	// GetMulti as memcache should have worked.
	nds.SetDatastoreGetMulti(func(c appengine.Context,
		keys []*datastore.Key, vals interface{}) error {
		if len(keys) != 0 {
			return errors.New("should not be called")
		}
		return nil
	})
	defer func() {
		nds.SetDatastoreGetMulti(datastore.GetMulti)
	}()
	tes := make([]testEntity, len(entities))
	if err := nds.GetMulti(c, keys, tes); err != nil {
		t.Fatal(err)
	}
}

func TestGetMultiNoKeys(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}

	if err := nds.GetMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}
}

func TestGetMultiInterfaceError(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Get from datastore.
	// No errors expected.
	response := []interface{}{&testEntity{}, &testEntity{}}

	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if te, ok := response[i].(*testEntity); ok {
			if te.IntVal != i+1 {
				t.Fatal("incorrect IntVal")
			}
		} else {
			t.Fatal("incorrect type")
		}
	}

	// Get from cache.
	// Errors expected.
	response = []interface{}{&testEntity{}, testEntity{}}
	if err := nds.GetMulti(c, keys, response); err == nil {
		t.Fatal("expected invalid entity type error")
	}
}

// This is just used to ensure interfaces don't currently work.
type readerTestEntity struct {
	IntVal int
}

func (rte readerTestEntity) Read(p []byte) (n int, err error) {
	return 1, nil
}

var _ io.Reader = readerTestEntity{}

func newReaderTestEntity() io.Reader {
	return readerTestEntity{}
}

func TestGetArgs(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	if err := nds.Get(c, nil, &testEntity{}); err == nil {
		t.Fatal("expected error for nil key")
	}

	key := datastore.NewKey(c, "Entity", "", 1, nil)
	if err := nds.Get(c, key, nil); err == nil {
		t.Fatal("expected error for nil value")
	}

	if err := nds.Get(c, key, datastore.PropertyList{}); err == nil {
		t.Fatal("expected error for datastore.PropertyList")
	}

	if err := nds.Get(c, key, testEntity{}); err == nil {
		t.Fatal("expected error for struct")
	}

	rte := newReaderTestEntity()
	if err := nds.Get(c, key, rte); err == nil {
		t.Fatal("expected error for interface")
	}
}

func TestGetMultiArgs(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	key := datastore.NewKey(c, "Entity", "", 1, nil)
	keys := []*datastore.Key{key}
	val := testEntity{}
	if err := nds.GetMulti(c, keys, nil); err == nil {
		t.Fatal("expected error for nil vals")
	}
	structVals := []testEntity{val}
	if err := nds.GetMulti(c, nil, structVals); err == nil {
		t.Fatal("expected error for nil keys")
	}

	if err := nds.GetMulti(c, keys, []testEntity{}); err == nil {
		t.Fatal("expected error for unequal keys and vals")
	}

	if err := nds.GetMulti(c, keys, datastore.PropertyList{}); err == nil {
		t.Fatal("expected error for propertyList")
	}

	rte := newReaderTestEntity()
	if err := nds.GetMulti(c, keys, []io.Reader{rte}); err == nil {
		t.Fatal("expected error for interface")
	}
}

func TestGetSliceProperty(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVals []int64
	}

	key := datastore.NewKey(c, "Entity", "", 1, nil)
	intVals := []int64{0, 1, 2, 3}
	val := &testEntity{intVals}

	if _, err := nds.Put(c, key, val); err != nil {
		t.Fatal(err)
	}

	// Get from datastore.
	newVal := &testEntity{}
	if err := nds.Get(c, key, newVal); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(val.IntVals, intVals) {
		t.Fatal("slice properties not equal", val.IntVals)
	}

	// Get from memcache.
	newVal = &testEntity{}
	if err := nds.Get(c, key, newVal); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(val.IntVals, intVals) {
		t.Fatal("slice properties not equal", val.IntVals)
	}
}

func TestGetMultiNoPropertyList(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	keys := []*datastore.Key{datastore.NewKey(c, "Test", "", 1, nil)}
	pl := datastore.PropertyList{datastore.Property{}}

	if err := nds.GetMulti(c, keys, pl); err == nil {
		t.Fatal("expecting no PropertyList error")
	}
}

func TestGetMultiNonStruct(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	keys := []*datastore.Key{datastore.NewKey(c, "Test", "", 1, nil)}
	vals := []int{12}

	if err := nds.GetMulti(c, keys, vals); err == nil {
		t.Fatal("expecting unsupported vals type")
	}
}

// TestGetMemcacheFail makes sure that memcache failure does not stop Get from
// working.
func TestGetMemcacheFail(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	key := datastore.NewKey(c, "Entity", "", 1, nil)
	val := &testEntity{3}

	if _, err := nds.Put(c, key, val); err != nil {
		t.Fatal(err)
	}

	nds.SetMemcacheAddMulti(func(c appengine.Context,
		items []*memcache.Item) error {
		return errors.New("expected memcache.AddMulti error")
	})
	nds.SetMemcacheCompareAndSwapMulti(func(c appengine.Context,
		items []*memcache.Item) error {
		return errors.New("expected memcache.ComapreAndSwap error")
	})
	nds.SetMemcacheGetMulti(func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		return nil, errors.New("expected memcache.GetMulti error")
	})
	nds.SetMemcacheSetMulti(func(c appengine.Context,
		items []*memcache.Item) error {
		return errors.New("expected memcache.SetMulti error")
	})

	defer func() {
		nds.SetMemcacheAddMulti(memcache.AddMulti)
		nds.SetMemcacheCompareAndSwapMulti(memcache.CompareAndSwapMulti)
		nds.SetMemcacheGetMulti(memcache.GetMulti)
		nds.SetMemcacheSetMulti(memcache.SetMulti)
	}()

	retVal := &testEntity{}
	if err := nds.Get(c, key, retVal); err != nil {
		t.Fatal(err)
	}
	if val.IntVal != retVal.IntVal {
		t.Fatal("val and retVal not equal")
	}
}

func TestGetMultiDatastoreFail(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	nds.SetDatastoreGetMulti(func(c appengine.Context,
		keys []*datastore.Key, vals interface{}) error {
		return errors.New("expected datastore.GetMulti error")
	})
	defer func() {
		nds.SetDatastoreGetMulti(datastore.GetMulti)
	}()

	// Get from datastore.
	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err == nil {
		t.Fatal("expected GetMulti to fail")
	}
}

func TestGetMultiMemcacheCorrupt(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Charge memcache.
	if err := nds.GetMulti(c, keys, make([]testEntity, len(keys))); err != nil {
		t.Fatal(err)
	}

	nds.SetMemcacheGetMulti(func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		items, err := memcache.GetMulti(c, keys)

		// Skip over lockMemcaheKeys GetMulti.
		if len(keys) != 0 {
			// Corrupt second item.
			items[keys[1]].Value = []byte("corrupt string")
		}

		return items, err
	})
	defer func() {
		nds.SetMemcacheGetMulti(memcache.GetMulti)
	}()

	// Get from datastore.
	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(keys); i++ {
		if entities[i].IntVal != response[i].IntVal {
			t.Fatal("IntVal not equal")
		}
	}
}

func TestGetMultiMemcacheFlagCorrupt(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Charge memcache.
	if err := nds.GetMulti(c, keys, make([]testEntity, len(keys))); err != nil {
		t.Fatal(err)
	}

	nds.SetMemcacheGetMulti(func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		items, err := memcache.GetMulti(c, keys)

		// Skip over lockMemcaheKeys GetMulti.
		if len(keys) != 0 {
			// Corrupt second item.
			items[keys[1]].Flags = 56
		}

		return items, err
	})
	defer func() {
		nds.SetMemcacheGetMulti(memcache.GetMulti)
	}()

	// Get from datastore.
	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(keys); i++ {
		if entities[i].IntVal != response[i].IntVal {
			t.Fatal("IntVal not equal")
		}
	}
}

func TestGetMultiLockMemcacheFailure(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	countLock := &sync.Mutex{}
	count := 0
	nds.SetMemcacheGetMulti(func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		items, err := memcache.GetMulti(c, keys)
		if count == 1 {
			items, err = nil, errors.New("expected lock error")
		}
		countLock.Lock()
		count++
		countLock.Unlock()
		return items, err
	})
	defer func() {
		nds.SetMemcacheGetMulti(memcache.GetMulti)
	}()

	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(keys); i++ {
		if entities[i].IntVal != response[i].IntVal {
			t.Fatal("IntVal not equal")
		}
	}
}
