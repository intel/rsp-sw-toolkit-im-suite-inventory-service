
STACK_NAME ?= Inventory-Suite-Dev
SERVICE_NAME ?= inventory
PROJECT_NAME ?= inventory-service

default: build

build::
	$(MAKE) -C .. $(PROJECT_NAME)

iterate::
	docker service scale $(STACK_NAME)_$(SERVICE_NAME)=0 -d
	$(MAKE) build
	docker service scale $(STACK_NAME)_$(SERVICE_NAME)=1 -d
	while [ -z `docker ps -qf name=$(STACK_NAME)_$(SERVICE_NAME).1` ]; \
	do \
		echo "Waiting for $(SERVICE_NAME) to start..."; \
		sleep 1;\
	done
	$(MAKE) tail

tail::
	docker logs -f `docker ps -qf name=$(STACK_NAME)_$(SERVICE_NAME).1`
