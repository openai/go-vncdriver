#include <stdarg.h>
#include <Python.h>
#include "structmember.h"

#define NPY_NO_DEPRECATED_API NPY_1_7_API_VERSION
#include <numpy/arrayobject.h>

typedef enum {
  NOTSET = 0,
  DEBUG = 10,
  INFO = 20,
  WARNING = 30,
  ERROR = 40,
  CRITICAL = 50,
} logging_level_e;

static PyObject *logger;

// First-time import the logging module, or throw an exception trying
int logger_init(void) {
  PyObject *logging_module = NULL;

  // python: import logging
  if (NULL == logger) {
    logging_module = PyImport_ImportModuleNoBlock("logging");
    if (NULL == logging_module) {
      PyErr_SetString(PyExc_ImportError, "Failed to import 'logging' module");
      return -1;
    }
  }

  return 0;
}

void logger_str(logging_level_e level, const char *format, ...) {
  char buf[1024];  // Should be enough for anybody...
  va_list(args);
  va_start(args, format);
  vsnprintf(buf, sizeof(buf), format, args);
  va_end(args);
  PyObject_CallMethod(logger, "log", "is", level, buf);
}

typedef struct {
      PyObject_HEAD
      /* Type-specific fields go here. */
} go_vncdriver_VNCSession_object;

static PyObject *go_vncdriver_Error;

/* Go functions exposed directly to Python */
// PyObject * GoVNCDriver_VNCSession_peek(PyObject *, PyObject *);
// PyObject * GoVNCDriver_VNCSession_flip(PyObject *, PyObject *);
PyObject * GoVNCDriver_VNCSession_step(PyObject *, PyObject *);
PyObject * GoVNCDriver_VNCSession_close(PyObject *, PyObject *, PyObject *);
PyObject * GoVNCDriver_VNCSession_render(PyObject *, PyObject *, PyObject *);
PyObject * GoVNCDriver_VNCSession_connect(PyObject *, PyObject *, PyObject *);
PyObject * GoVNCDriver_VNCSession_update(PyObject *, PyObject *, PyObject *);

/* Go functions which are called only from C */
int GoVNCDriver_VNCSession_c_init(go_vncdriver_VNCSession_object *);
void GoVNCDriver_VNCSession_c_dealloc(go_vncdriver_VNCSession_object *);

/* Global functions */
PyObject * GoVNCDriver_setup(PyObject *, PyObject *);

/* end go functions */

void PyErr_SetGoVNCDriverError(char* msg) {
    PyErr_SetString(go_vncdriver_Error, msg);
    free(msg);
}

PyObject *GoPyArray_SimpleNew(int nd, npy_intp* dims, int typenum) {
    return PyArray_SimpleNew(nd, dims, typenum);
}

PyObject *GoPyArray_SimpleNewFromData(int nd, npy_intp* dims, int typenum, void *data) {
  return PyArray_SimpleNewFromData(nd, dims, typenum, data);
}

/* VNCSession object */

static void
go_vncdriver_VNCSession_dealloc(go_vncdriver_VNCSession_object* self)
{
    GoVNCDriver_VNCSession_c_dealloc(self);
#if PY_MAJOR_VERSION >= 3
    Py_TYPE(self)->tp_free((PyObject*)self);
#else
    self->ob_type->tp_free((PyObject*)self);
#endif
}

static int
go_vncdriver_VNCSession_init(go_vncdriver_VNCSession_object *self, PyObject *args, PyObject *kwds)
{
    static char *kwlist[] = {NULL};

    // No args!
    if (!PyArg_ParseTupleAndKeywords(args, kwds, "", kwlist))
        return -1;

    int res = GoVNCDriver_VNCSession_c_init(self);
    if (res == -1) {
        return -1;
    }

    return 0;
}

static PyMemberDef go_vncdriver_VNCSession_members[] = {
    {NULL}  /* Sentinel */
};

static PyMethodDef go_vncdriver_VNCSession_methods[] = {
  {"connect", (PyCFunction)GoVNCDriver_VNCSession_connect, METH_VARARGS|METH_KEYWORDS, "Connect an index to a new remote"},
  {"close", (PyCFunction)GoVNCDriver_VNCSession_close, METH_VARARGS|METH_KEYWORDS, "Closes the connection"},
  //  {"flip", (PyCFunction)GoVNCDriver_VNCSession_flip, METH_NOARGS, "Flips to the most recently updates screen"},
  //  {"peek", (PyCFunction)GoVNCDriver_VNCSession_peek, METH_NOARGS, "Peek at the last returned screen"},
  {"render", (PyCFunction)GoVNCDriver_VNCSession_render, METH_VARARGS|METH_KEYWORDS, "Render the screen"},
  {"step", (PyCFunction)GoVNCDriver_VNCSession_step, METH_O, "Perform actions and then flip"},
  {"update", (PyCFunction) GoVNCDriver_VNCSession_update, METH_VARARGS|METH_KEYWORDS, "Update the connection options"},
  {NULL}  /* Sentinel */
};

// https://docs.python.org/2.7/extending/newtypes.html
PyTypeObject go_vncdriver_VNCSession_type = {
    PyVarObject_HEAD_INIT(NULL, 0)
    "go_vncdriver.VNCSession",             /*tp_name*/
    sizeof(go_vncdriver_VNCSession_object), /*tp_basicsize*/
};

// Needed because CGo can't access static variables
PyObject *get_go_vncdriver_VNCSession_type() {
  return (PyObject *) &go_vncdriver_VNCSession_type;
}

static PyMethodDef go_vncdriver_module_methods[] = {
    {NULL, NULL, 0, NULL}
};

// https://docs.python.org/3/howto/cporting.html#module-initialization-and-state
struct go_vncdriver_module_state {
    PyObject *error;
};

#if PY_MAJOR_VERSION >= 3
#define GETSTATE(m) ((struct go_vncdriver_module_state*)PyModule_GetState(m))
#else
#define GETSTATE(m) (&_state)
static struct go_vncdriver_module_state _state;
#endif


static PyObject *
error_out(PyObject *m) {
    struct go_vncdriver_module_state *st = GETSTATE(m);
    PyErr_SetString(go_vncdriver_Error, "something bad happened");
    return NULL;
}

#if PY_MAJOR_VERSION >= 3

static struct PyModuleDef go_vncdriver_module = {
        PyModuleDef_HEAD_INIT,
        "go_vncdriver",
        NULL,
        sizeof(struct go_vncdriver_module_state),
        go_vncdriver_module_methods,
        NULL,
        NULL,  // No traverse for now
        NULL,  // No clear for now
        NULL
};

#define INITERROR return NULL

PyMODINIT_FUNC
PyInit_go_vncdriver(void)

#else
#define INITERROR return

void
initgo_vncdriver(void)
#endif
{
#if PY_MAJOR_VERSION >= 3
    PyObject *module = PyModule_Create(&go_vncdriver_module);
#else
    PyObject *module = Py_InitModule("go_vncdriver", go_vncdriver_module_methods);
#endif

    if (logger_init() < 0)
        INITERROR;

    if (module == NULL)
        INITERROR;
    struct go_vncdriver_module_state *st = GETSTATE(module);

    go_vncdriver_Error = PyErr_NewException("go_vncdriver.Error", NULL, NULL);
    if (go_vncdriver_Error == NULL) {
        Py_DECREF(module);
        INITERROR;
    }

    Py_INCREF(go_vncdriver_Error);
    PyModule_AddObject(module, "Error", go_vncdriver_Error);

    go_vncdriver_VNCSession_type.tp_dealloc = (destructor)go_vncdriver_VNCSession_dealloc;
    go_vncdriver_VNCSession_type.tp_flags = Py_TPFLAGS_DEFAULT | Py_TPFLAGS_BASETYPE;
    go_vncdriver_VNCSession_type.tp_doc = "VNCSession objects";
    go_vncdriver_VNCSession_type.tp_methods = go_vncdriver_VNCSession_methods;
    go_vncdriver_VNCSession_type.tp_members = go_vncdriver_VNCSession_members;
    go_vncdriver_VNCSession_type.tp_init = (initproc)go_vncdriver_VNCSession_init;
    go_vncdriver_VNCSession_type.tp_new = PyType_GenericNew;
    if (PyType_Ready(&go_vncdriver_VNCSession_type) < 0) {
        Py_DECREF(module);
        INITERROR;
    }

    Py_INCREF(&go_vncdriver_VNCSession_type);
    PyModule_AddObject(module, "VNCSession", (PyObject *) &go_vncdriver_VNCSession_type);

    // TODO: need to py3 this?
    import_array();

#if PY_MAJOR_VERSION >= 3
    return module;
#endif
}
