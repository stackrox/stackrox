# Performance Testing for StackRox - Quick Start

Please use `README.ipynb` for local developnment of performance tests. That README is designed to be easy executable.

### Pre-requirements to use `README.ipynb`

You must install Jupyter `bash` kernel. That can be done with the following commands:
```
pip install bash_kernel
python -m bash_kernel.install
```

After that, you can use VS Code with Jupyter plugin to easy render notebook and execute commands from it.

For that you can install the following plugins:
* https://marketplace.visualstudio.com/items?itemName=ms-toolsai.jupyter
* https://marketplace.visualstudio.com/items?itemName=ms-python.python

Then ensure that the Jupyter server is up and running, open the search bar by hitting `cmd+shift+p` and looking for `Jupyter: Select Interpreter to Start Jupyter Server`.
Once the server is up and running you should be able to configure it with a desired Kernel if not already set. In the top right look for what language is configured then select `Select Another Kernel` -> `Jupyter Kernel` -> `Bash`.