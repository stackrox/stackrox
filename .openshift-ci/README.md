This folder contains hooks to run OpenShift CI jobs. 

## Workflow aliases for test/format/lint

alias osci='cdrox; cd .openshift-ci'
alias osci-format='osci; ack -f --python | entr black .'
alias osci-lint='osci; ack -f --python | entr pylint --rcfile .pylintrc *.py tests'
alias osci-test='osci; ack -f --python --shell | entr python -m unittest discover'
