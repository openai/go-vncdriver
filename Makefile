.PHONY: build no_gl clean upload install test

build:
	./build.py

no_gl:
	GO_VNCDRIVER_NOGL=true ./build.py

install:
	pip install -r requirements.txt

upload: clean
	rm -rf dist go_vncdriver.egg-info
	python setup.py sdist
	twine upload dist/*
	rm -rf dist

clean:
	rm -rf *.so *.h *~ build

test:
	python -c 'import go_vncdriver; go_vncdriver.VNCSession()'
