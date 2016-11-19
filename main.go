package main

/*
#define Py_LIMITED_API
#include <Python.h>
#include <stdlib.h>
#define NPY_NO_DEPRECATED_API NPY_1_7_API_VERSION
#include <numpy/arrayobject.h>

PyObject *get_go_vncdriver_VNCSession_type();
PyObject *GoPyArray_SimpleNew(int nd, npy_intp* dims, int typenum);
PyObject *GoPyArray_SimpleNewFromData(int nd, npy_intp* dims, int typenum, void *data);
void PyErr_SetGoVNCDriverError(char* msg);

// Workaround missing variadic function support
// https://github.com/golang/go/issues/975
static int PyArg_ParseTuple_list_O(PyObject *args, PyObject **a) {
    return PyArg_ParseTuple(args, "O", &PyList_Type, a);
}

static int PyArg_ParseTuple_connect(PyObject *args, PyObject *kwds, char **name, char **address, char **password, char **encoding, int *quality_level, int *compress_level, int *fine_quality_level, int *subsample_level, unsigned long *start_timeout, PyObject **subscription) {
    static char *kwlist[] = {"name", "address", "password", "encoding", "quality_level", "compress_level", "fine_quality_level", "subsample_level", "start_timeout", "subscription", NULL};
    return PyArg_ParseTupleAndKeywords(args, kwds, "ss|ssiiiikO", kwlist, name, address, password, encoding, quality_level, compress_level, fine_quality_level, subsample_level, start_timeout, subscription);
}

static int PyArg_ParseTuple_close(PyObject *args, PyObject *kwds, char **name) {
    static char *kwlist[] = {"name", NULL};
    *name = "";
    return PyArg_ParseTupleAndKeywords(args, kwds, "|s", kwlist, name);
}

static int PyArg_ParseTuple_name(PyObject *args, PyObject *kwds, char **name) {
    static char *kwlist[] = {"name", NULL};
    return PyArg_ParseTupleAndKeywords(args, kwds, "s", kwlist, name);
}

static int PyArg_ParseTuple_render(PyObject *args, PyObject *kwds, char **name, int *close) {
    static char *kwlist[] = {"name", "close", NULL};
    return PyArg_ParseTupleAndKeywords(args, kwds, "s|i", kwlist, name, close);
}

static int PyArg_ParseTuple_update(PyObject *args, PyObject *kwds, char **name, PyObject **subscription) {
    static char *kwlist[] = {"name", "subscription", NULL};
    return PyArg_ParseTupleAndKeywords(args, kwds, "sO", kwlist, name, subscription);
}

static PyObject *PyObject_CallFunctionObjArgs_1(PyObject *callable, PyObject *a) {
    return PyObject_CallFunctionObjArgs(callable, a, NULL);
}

typedef struct {
      PyObject_HEAD
      PyObject *addresses;
} go_vncdriver_VNCSession_object;

// Can't access macros through cgo
static void go_vncdriver_decref(PyObject *obj) {
    Py_DECREF(obj);
}

static void go_vncdriver_incref(PyObject *obj) {
    Py_INCREF(obj);
}
*/
import "C"
import (
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/juju/errors"
	"github.com/op/go-logging"
	"github.com/openai/go-vncdriver/gymvnc"
	"github.com/openai/go-vncdriver/vncclient"
)

var (
	log     = logging.MustGetLogger("go_vncdriver")
	Py_None = &C._Py_NoneStruct

	emptyString = C.CString("")

	vncUpdatesN          *C.PyObject = nil
	vncUpdatesPixels     *C.PyObject = nil
	vncUpdatesRectangles *C.PyObject = nil
	vncUpdatesBytes      *C.PyObject = nil

	setup sync.Once
)

func setupOnce() {
	// Must hold the GIL when we init these. Thus don't need
	// Go-level locking as well.
	vncUpdatesN = C.PyUnicode_FromString(C.CString("stats.vnc.updates.n"))
	vncUpdatesPixels = C.PyUnicode_FromString(C.CString("stats.vnc.updates.pixels"))
	vncUpdatesRectangles = C.PyUnicode_FromString(C.CString("stats.vnc.updates.rectangles"))
	vncUpdatesBytes = C.PyUnicode_FromString(C.CString("stats.vnc.updates.bytes"))

	gymvnc.ConfigureLogging()
}

//export GoVNCDriver_VNCSession_c_init
func GoVNCDriver_VNCSession_c_init(self *C.go_vncdriver_VNCSession_object) C.int {
	setup.Do(setupOnce)

	batch := gymvnc.NewVNCBatch()
	info := sessionInfo{
		batch: batch,
		names: map[string]*C.char{},

		screenNumpy: map[string]map[*gymvnc.Screen]*C.PyObject{},
	}

	if !info.preallocatePythonObjects() {
		return C.int(-1)
	}

	batchLock.Lock()
	defer batchLock.Unlock()

	ptr := uintptr(unsafe.Pointer(self))
	batchMgr[ptr] = info

	return C.int(0)
}

//export GoVNCDriver_VNCSession_connect
func GoVNCDriver_VNCSession_connect(self, args, kwds *C.PyObject) *C.PyObject {
	batchLock.Lock()
	defer batchLock.Unlock()

	// Just store this as a property on self?
	ptr := uintptr(unsafe.Pointer(self))
	info, ok := batchMgr[ptr]
	if !ok {
		setError(errors.ErrorStack(errors.New("VNCSession is already closed")))
		return nil
	}

	_ = info

	nameC := new(*C.char)
	addressC := new(*C.char)
	passwordC := new(*C.char)
	encodingC := new(*C.char)
	qualityLevelC := new(C.int)
	compressLevelC := new(C.int)
	fineQualityLevelC := new(C.int)
	subsampleLevelC := new(C.int)
	startTimeoutC := new(C.ulong)
	subscriptionPy := new(*C.PyObject)

	*compressLevelC = C.int(-1)
	*qualityLevelC = C.int(-1)
	*fineQualityLevelC = C.int(-1)
	*subsampleLevelC = C.int(-1)

	if C.PyArg_ParseTuple_connect(args, kwds, nameC, addressC, passwordC, encodingC, qualityLevelC, compressLevelC, fineQualityLevelC, subsampleLevelC, startTimeoutC, subscriptionPy) == 0 {
		return nil
	}

	name := C.GoString(*nameC)
	address := C.GoString(*addressC)
	password := C.GoString(*passwordC)
	encoding := C.GoString(*encodingC)
	qualityLevel := int(*qualityLevelC)
	compressLevel := int(*compressLevelC)
	fineQualityLevel := int(*fineQualityLevelC)
	subsampleLevel := int(*subsampleLevelC)
	startTimeout := int(*startTimeoutC)
	subscription, ok := convertSubscriptionPy(*subscriptionPy)
	if !ok {
		return nil
	}

	if _, ok := info.names[name]; ok {
		log.Infof("disconnecting existing connection %s", name)
		info.close(name)
	}

	if password == "" {
		// our default password!
		password = "openai"
	}

	err := info.batch.Open(name, gymvnc.VNCSessionConfig{
		Address:  address,
		Password: password,
		Encoding: encoding,

		QualityLevel:     qualityLevel,
		CompressLevel:    compressLevel,
		FineQualityLevel: fineQualityLevel,
		SubsampleLevel:   subsampleLevel,
		StartTimeout:     time.Duration(startTimeout) * time.Second,

		Subscription: subscription,
	})
	if err != nil {
		setError(errors.ErrorStack(err))
		return nil
	}
	info.open(name)

	// errCh := make(chan error, 10)
	// done := make(chan bool)
	// batch, err := gymvnc.NewVNCSession(, )
	// if err != nil {
	// 	close(done)
	// 	setError(errors.ErrorStack(err))
	// 	return C.int(-1)
	// }

	C.go_vncdriver_incref(Py_None)
	return Py_None
}

//export GoVNCDriver_VNCSession_step
func GoVNCDriver_VNCSession_step(self, actionDict *C.PyObject) (rep *C.PyObject) {
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		stack := debug.Stack()
	// 		setError(string(stack))
	// 		rep = nil
	// 	}
	// }()

	batchLock.Lock()
	defer batchLock.Unlock()

	ptr := uintptr(unsafe.Pointer(self))
	info, ok := batchMgr[ptr]
	if !ok {
		setError("VNCSession is closed")
		return nil
	}

	// Iterate over all active sessions
	batchEvents := map[string][]gymvnc.VNCEvent{}
	ok = true

	for name, nameC := range info.names {
		events := []gymvnc.VNCEvent{}
		action := C.PyDict_GetItemString(actionDict, nameC)
		if C.PyErr_Occurred() != nil {
			return nil
		}

		if action != nil {
			// Iterate over all events in the action
			eventsIter := C.PyObject_GetIter(action)
			for eventPy := C.PyIter_Next(eventsIter); ok && eventPy != nil; eventPy = C.PyIter_Next(eventsIter) {
				event, convertOk := convertEventPy(eventPy)
				if !convertOk {
					ok = false
				}
				C.go_vncdriver_decref(eventPy)

				events = append(events, event)
			}
			C.go_vncdriver_decref(eventsIter)
		}
		if !ok {
			if C.PyErr_Occurred() == nil {
				setError(errors.ErrorStack(errors.Errorf("BUG: unpacking actions failed for %s, but no Python error was set", name)))
				return nil
			}
			return nil
		}

		batchEvents[name] = events
	}

	// Put together the Python objects
	observationN, updatesN, errN := info.batch.Step(batchEvents)
	if ok := info.populateScreenPyDict(observationN); !ok {
		return nil
	}
	if ok := info.populateInfoPyDict(updatesN); !ok {
		return nil
	}
	if ok := info.populateErrorPyDict(errN); !ok {
		return nil
	}

	// Actually, let's make this the user's responsibility, so
	// they don't have disappearing names.
	//
	// // Cleanup anything that broke
	// for name, err := range errN {
	// 	if err != nil {
	// 		info.close(name)
	// 	}
	// }

	// Ownership will transfer away when we return
	C.go_vncdriver_incref(info.screenInfoErrPytuple)
	return info.screenInfoErrPytuple
}

//export GoVNCDriver_VNCSession_update
func GoVNCDriver_VNCSession_update(self, args, kwds *C.PyObject) *C.PyObject {
	batchLock.Lock()
	defer batchLock.Unlock()

	ptr := uintptr(unsafe.Pointer(self))
	info, ok := batchMgr[ptr]
	if !ok {
		setError("VNCSession is closed")
		return nil
	}

	nameC := new(*C.char)
	subscriptionPy := new(*C.PyObject)
	if C.PyArg_ParseTuple_update(args, kwds, nameC, subscriptionPy) == 0 {
		return nil
	}
	name := C.GoString(*nameC)
	subscription, ok := convertSubscriptionPy(*subscriptionPy)
	if !ok {
		return nil
	}

	err := info.batch.SetSubscription(name, subscription)
	if err != nil {
		setError(errors.ErrorStack(err))
		return nil
	}

	C.go_vncdriver_incref(Py_None)
	return Py_None
}

var (
	batchMgr  = map[uintptr]sessionInfo{}
	batchLock sync.Mutex
)

func unpackStrings(strings *C.PyObject) ([]string, error) {
	strs := []string{}
	size := C.PyList_Size(strings)
	for i := 0; i < int(size); i++ {
		// Look at the i'th item and convert to a Go string
		listItem := C.PyList_GetItem(strings, C.Py_ssize_t(i))
		if listItem == nil {
			return nil, errors.New("list lookup failed")
		}
		unicodeObj := C.PyUnicode_FromObject(listItem)
		if unicodeObj == nil {
			return nil, errors.New("could not convert item to unicode")
		}
		byteObj := C.PyUnicode_AsASCIIString(unicodeObj)
		if byteObj == nil {
			return nil, errors.New("could not convert unicode to ascii string")
		}
		strObj := C.PyBytes_AsString(byteObj)
		if strObj == nil {
			return nil, errors.New("could not convert bytes to C string")
		}
		str := C.GoString(strObj)
		strs = append(strs, str)
	}
	return strs, nil
}

//export GoVNCDriver_VNCSession_render
func GoVNCDriver_VNCSession_render(self, args, kwds *C.PyObject) *C.PyObject {
	batchLock.Lock()
	defer batchLock.Unlock()

	// parse name argument
	nameC := new(*C.char)
	closeC := new(C.int)
	if C.PyArg_ParseTuple_render(args, kwds, nameC, closeC) == 0 {
		return nil
	}
	name := C.GoString(*nameC)
	close := *closeC != C.int(0)

	ptr := uintptr(unsafe.Pointer(self))
	info, ok := batchMgr[ptr]
	if !ok {
		setError(errors.ErrorStack(errors.New("VNCSession is already closed")))
		return nil
	}

	info.initRenderer(name)

	err := info.batch.Render(name, close)
	if err != nil {
		// reportBestError(info.batch, err)
		setError(errors.ErrorStack(err))
		return nil
	}

	C.go_vncdriver_incref(Py_None)
	return Py_None
}

// func reportBestError(batch *gymvnc.VNCBatch, err error) {
// 	var report string

// 	checkErr := batch.Check()
// 	if checkErr != nil {
// 		report = fmt.Sprintf("Error: %s\n\nOriginal error: %s", errors.ErrorStack(err), errors.ErrorStack(checkErr))
// 	} else {
// 		report = fmt.Sprintf("Error: %s", errors.ErrorStack(err))
// 	}

// 	setError(report)
// }

func GoString_FromPyString(t *C.PyObject) (string, bool) {
	unicodePystr := C.PyUnicode_FromObject(t)
	if unicodePystr == nil {
		setError(errors.ErrorStack(errors.New("could not convert to unicode")))
		return "", false
	}
	bytePystr := C.PyUnicode_AsASCIIString(unicodePystr)
	if bytePystr == nil {
		setError(errors.ErrorStack(errors.New("could not encode to ascii")))
		return "", false
	}
	typePystr := C.PyBytes_AsString(bytePystr)
	if typePystr == nil {
		setError(errors.ErrorStack(errors.New("could not convert bytes to cstring")))
		return "", false
	}
	return C.GoString(typePystr), true
}

func convertSubscriptionPy(subscriptionPy *C.PyObject) (regions []gymvnc.Region, ok bool) {
	ok = true

	if subscriptionPy == nil {
		return
	}

	// Iterate over all subscription
	subscriptionIter := C.PyObject_GetIter(subscriptionPy)
	if subscriptionIter == nil {
		ok = false
		return
	}

	for itemPy := C.PyIter_Next(subscriptionIter); ok && itemPy != nil; itemPy = C.PyIter_Next(subscriptionIter) {
		var x, y, width, height int

		if ok {
			x, ok = getIntFromTuple(itemPy, 0)
		}
		if ok {
			width, ok = getIntFromTuple(itemPy, 1)
		}
		if ok {
			y, ok = getIntFromTuple(itemPy, 2)
		}
		if ok {
			height, ok = getIntFromTuple(itemPy, 3)
		}

		C.go_vncdriver_decref(itemPy)
		regions = append(regions, gymvnc.Region{
			X:      uint16(x),
			Y:      uint16(y),
			Width:  uint16(width),
			Height: uint16(height),
		})
	}
	C.go_vncdriver_decref(subscriptionIter)

	return
}

// Sets python error
func convertEventPy(eventPy *C.PyObject) (event gymvnc.VNCEvent, ok bool) {
	// eventPy: ("PointerEvent", x, y, buttonmask) or ("KeyEvent", key, down)

	// if PyTuple_Check(eventPy) == 0 {
	// 	setError("event was not a tuple")
	// 	ok = false
	// 	return
	// }
	t := C.PyTuple_GetItem(eventPy, C.Py_ssize_t(0))
	if t == nil {
		C.PyErr_Clear()
		repr, isOk := PyObject_Repr(eventPy)
		if !isOk {
			return
		}
		setError(fmt.Sprintf("Expected non-empty tuple rather than: %s", repr))
		return
	}

	eventType, isOk := GoString_FromPyString(t)
	if !isOk {
		return
	}

	if eventType == "PointerEvent" {
		x, isOk := getIntFromTuple(eventPy, 1)
		if !isOk {
			return
		}

		y, isOk := getIntFromTuple(eventPy, 2)
		if !isOk {
			return
		}

		mask, isOk := getIntFromTuple(eventPy, 3)
		if !isOk {
			return
		}

		event = gymvnc.PointerEvent{
			Mask: vncclient.ButtonMask(mask),
			X:    uint16(x),
			Y:    uint16(y),
		}
	} else if eventType == "KeyEvent" {
		keysym, isOk := getIntFromTuple(eventPy, 1)
		if !isOk {
			return
		}

		down, isOk := getBoolFromTuple(eventPy, 2)
		if !isOk {
			return
		}

		event = gymvnc.KeyEvent{
			Keysym: uint32(keysym),
			Down:   down,
		}
	} else {
		setError(fmt.Sprintf("invalid event type: %s", eventType))
	}

	ok = true
	return
}

func getIntFromTuple(eventPy *C.PyObject, i int) (int, bool) {
	iPyint := C.PyTuple_GetItem(eventPy, C.Py_ssize_t(i))
	if iPyint == nil {
		return 0, false
	}

	tup := C.PyLong_AsLong(iPyint)
	if tup == -1 && C.PyErr_Occurred() != nil {
		return 0, false
	}

	return int(tup), true
}

func getBoolFromTuple(eventPy *C.PyObject, i int) (bool, bool) {
	iPyint := C.PyTuple_GetItem(eventPy, C.Py_ssize_t(i))
	if iPyint == nil {
		return false, false
	}

	t := C.PyObject_IsTrue(iPyint)
	if t == -1 {
		return false, false
	}

	if t == 1 {
		return true, true
	} else {
		return false, true
	}
}

func PyObject_Repr(obj *C.PyObject) (string, bool) {
	res := C.PyObject_Repr(obj)
	if res == nil {
		return "", false
	}

	unicodeObj := C.PyUnicode_FromObject(res)
	if unicodeObj == nil {
		return "", false
	}

	byteObj := C.PyUnicode_AsASCIIString(unicodeObj)
	if byteObj == nil {
		return "", false
	}

	strObj := C.PyBytes_AsString(byteObj)
	if strObj == nil {
		return "", false
	}

	str := C.GoString(strObj)
	return str, true
}

//export GoVNCDriver_VNCSession_close
func GoVNCDriver_VNCSession_close(self, args, kwds *C.PyObject) *C.PyObject {
	nameC := new(*C.char)
	if C.PyArg_ParseTuple_close(args, kwds, nameC) == 0 {
		return nil
	}
	name := C.GoString(*nameC)

	if name == "" {
		log.Debug("closing entire VNCSession")
		cast := (*C.go_vncdriver_VNCSession_object)(unsafe.Pointer(self))
		GoVNCDriver_VNCSession_c_dealloc(cast)
	} else {
		log.Debugf("closing %s connection in VNCSession", name)

		batchLock.Lock()
		defer batchLock.Unlock()

		ptr := uintptr(unsafe.Pointer(self))
		info, ok := batchMgr[ptr]
		if !ok {
			setError(errors.ErrorStack(errors.New("VNCSession is already closed")))
			return nil
		}

		info.close(name)
	}

	C.go_vncdriver_incref(Py_None)
	return Py_None
}

//export GoVNCDriver_VNCSession_c_dealloc
func GoVNCDriver_VNCSession_c_dealloc(self *C.go_vncdriver_VNCSession_object) {
	batchLock.Lock()
	defer batchLock.Unlock()

	ptr := uintptr(unsafe.Pointer(self))
	info, ok := batchMgr[ptr]
	if !ok {
		return
	}

	for name := range info.names {
		info.close(name)
	}

	var gstate C.PyGILState_STATE
	gstate = C.PyGILState_Ensure()
	defer C.PyGILState_Release(gstate)

	if info.screenInfoErrPytuple != nil {
		C.go_vncdriver_decref(info.screenInfoErrPytuple)
	}
	for _, screenToNumpy := range info.screenNumpy {
		for _, screen := range screenToNumpy {
			C.go_vncdriver_decref(screen)
		}
	}

	// We do not own references to screenPyDict / infoPyDict / errPyDict
	delete(batchMgr, ptr)
}

func setError(str string) {
	C.PyErr_SetGoVNCDriverError(C.CString(str))
}

type sessionInfo struct {
	batch *gymvnc.VNCBatch

	names map[string]*C.char

	// return type: {...}, {...}, {...}
	screenPyDict *C.PyObject
	infoPyDict   *C.PyObject
	errPyDict    *C.PyObject

	screenInfoErrPytuple *C.PyObject
	screenNumpy          map[string]map[*gymvnc.Screen]*C.PyObject

	rendererSet bool
}

func (b *sessionInfo) open(name string) {
	b.names[name] = C.CString(name)
	b.screenNumpy[name] = map[*gymvnc.Screen]*C.PyObject{}
}

func (b *sessionInfo) close(name string) {
	b.batch.Close(name)

	nameC := b.names[name]
	C.free(unsafe.Pointer(nameC))
	delete(b.names, name)

	for _, screenNumpy := range b.screenNumpy[name] {
		C.go_vncdriver_decref(screenNumpy)
	}
	delete(b.screenNumpy, name)
}

// Sets the Python error for you
func (b *sessionInfo) populateScreenPyDict(screens map[string]*gymvnc.Screen) bool {
	C.PyDict_Clear(b.screenPyDict)

	for name, screen := range screens {
		var ary *C.PyObject
		if screen != nil {
			var ok bool
			screenToNumpy := b.screenNumpy[name]
			ary, ok = screenToNumpy[screen]
			if !ok {
				// allocate a new screen object, once
				dims := []C.npy_intp{C.npy_intp(screen.Height), C.npy_intp(screen.Width), 3}
				ary = C.GoPyArray_SimpleNewFromData(3, &dims[0], C.NPY_UINT8, unsafe.Pointer(&screen.Data[0]))
				screenToNumpy[screen] = ary
			}
		} else {
			ary = Py_None
		}

		if C.PyDict_SetItemString(b.screenPyDict, b.names[name], ary) == C.int(-1) {
			return false
		}
	}
	return true
}

func (b *sessionInfo) populateInfoPyDict(updateN map[string][]*vncclient.FramebufferUpdateMessage) bool {
	C.PyDict_Clear(b.infoPyDict)

	for name, update := range updateN {
		dict := C.PyDict_New()
		// Put the new dictionary into the info dict we're returning
		ok := C.PyDict_SetItemString(b.infoPyDict, b.names[name], dict)
		C.go_vncdriver_decref(dict)
		if ok != C.int(0) {
			return false
		}

		// Retain our reference!
		// C.go_vncdriver_incref(vncUpdatesN)
		updatePy := C.PyLong_FromLong(C.long(len(update)))
		ok = C.PyDict_SetItem(dict, vncUpdatesN, updatePy)
		C.go_vncdriver_decref(updatePy)
		if ok != C.int(0) {
			return false
		}

		// Count up number of pixels changed
		pixels := 0
		rectangles := 0
		bytes := 0 // TODO: should we just compute this from the byte stream directly?
		for _, updateI := range update {
			for _, rect := range updateI.Rectangles {
				pixels += int(rect.Width) * int(rect.Height)
				rectangles++
				// Each rectangle consists of X, Y,
				// Width, Height (each 2 bytes) and an
				// encoding.
				//
				// Technically, we should consider
				// including other control messages in
				// bytes, but this is ok for now.
				bytes += rect.Enc.Size() + 8
			}
		}

		// C.go_vncdriver_incref(vncUpdatesPixels)
		pixelsPy := C.PyLong_FromLong(C.long(pixels))
		ok = C.PyDict_SetItem(dict, vncUpdatesPixels, pixelsPy)
		C.go_vncdriver_decref(pixelsPy)
		if ok != C.int(0) {
			return false
		}

		// C.go_vncdriver_incref(vncUpdatesRectangles)
		rectsPy := C.PyLong_FromLong(C.long(rectangles))
		ok = C.PyDict_SetItem(dict, vncUpdatesRectangles, rectsPy)
		C.go_vncdriver_decref(rectsPy)
		if ok != C.int(0) {
			return false
		}

		// C.go_vncdriver_incref(vncUpdatesBytes)
		bytesPy := C.PyLong_FromLong(C.long(bytes))
		ok = C.PyDict_SetItem(dict, vncUpdatesBytes, bytesPy)
		C.go_vncdriver_decref(bytesPy)
		if ok != C.int(0) {
			return false
		}
	}

	return true
}

func (b *sessionInfo) populateErrorPyDict(errN map[string]error) bool {
	C.PyDict_Clear(b.errPyDict)

	for name, err := range errN {
		if err != nil {
			errC := C.CString(err.Error())
			errPy := C.PyUnicode_FromString(errC)
			C.free(unsafe.Pointer(errC))
			if errPy == nil {
				return false
			}

			ok := C.PyDict_SetItemString(b.errPyDict, b.names[name], errPy)
			C.go_vncdriver_decref(errPy)
			if ok != C.int(0) {
				return false
			}
		}
	}
	return true
}

// Preallocate all needed Python objects, so we don't need to generate
// a lot of garbage. (In practice the speedup here might be minimal,
// but does save a bit of code complexity.
func (b *sessionInfo) preallocatePythonObjects() bool {
	b.screenPyDict = C.PyDict_New()
	if b.screenPyDict == nil {
		return false
	}

	b.infoPyDict = C.PyDict_New()
	if b.infoPyDict == nil {
		return false
	}

	b.errPyDict = C.PyDict_New()
	if b.errPyDict == nil {
		return false
	}

	b.screenInfoErrPytuple = C.PyTuple_New(C.Py_ssize_t(3))
	if b.screenInfoErrPytuple == nil {
		return false
	}

	if i := C.PyTuple_SetItem(b.screenInfoErrPytuple, C.Py_ssize_t(0), b.screenPyDict); i != C.int(0) {
		return false
	}

	if i := C.PyTuple_SetItem(b.screenInfoErrPytuple, C.Py_ssize_t(1), b.infoPyDict); i != C.int(0) {
		return false
	}

	if i := C.PyTuple_SetItem(b.screenInfoErrPytuple, C.Py_ssize_t(2), b.errPyDict); i != C.int(0) {
		return false
	}

	return true
}

func main() {
}
