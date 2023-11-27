VENV := .dev-venv

ifdef CI
	PYCODESTYLE=pycodestyle
	AUTOPEP8=autopep8
	PYLINT=pylint
else
	PYCODESTYLE=$(VENV)/bin/pycodestyle
	AUTOPEP8=$(VENV)/bin/autopep8
	PYLINT=$(VENV)/bin/pylint
endif

.PHONY: all
all: style lint

$(VENV)/bin/activate: dev-requirements.txt
ifdef CI
	$(PYCODESTYLE) --version
	$(PYLINT) --version
else
	python3 -m venv $(VENV)
	$(VENV)/bin/pip3 --quiet --require-virtualenv --no-input --disable-pip-version-check install -r dev-requirements.txt
endif

.PHONY: style
style: $(VENV)/bin/activate
	$(PYCODESTYLE) --max-line-length=120 --exclude=$(VENV) .

.PHONY: fix-style
fix-style: $(VENV)/bin/activate
	$(AUTOPEP8) --in-place --exclude=$(VENV) *.py **/*.py

.PHONY: lint
lint: $(VENV)/bin/activate
	$(PYLINT) --rcfile .pylintrc --ignore=$(VENV) *.py **/*.py ../scripts/ci/jobs/*.py
