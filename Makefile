.PHONY: build clean upload install test

build:
	./build.sh

no_gl:
	./build.sh no_gl

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
	python -c 'import go_vncdriver; go_vncdriver.setup(); go_vncdriver.VNCSession(["localhost:5900"])'
