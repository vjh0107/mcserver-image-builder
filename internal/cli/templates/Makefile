REGISTRY := registry.example.com

.PHONY: all warm build push clean validate

all: warm build

warm:
	mcserver warm --all

build:
	mcserver build --all --registry $(REGISTRY)

push:
	mcserver build --all --registry $(REGISTRY) --push

validate:
	mcserver validate --all

clean:
	find . -name ".warm" -type d -exec rm -rf {} +
