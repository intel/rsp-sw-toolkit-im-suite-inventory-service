
STACK_NAME ?= Inventory-Suite-Dev
SERVICE_NAME ?= inventory
PROJECT_NAME ?= inventory-service

default: build

scale=docker service scale $(STACK_NAME)_$(SERVICE_NAME)=$1 $2

wait_for_service=	@printf "Waiting for $(SERVICE_NAME) service to$1..."; \
					while [  $2 -z `docker ps -qf name=$(STACK_NAME)_$(SERVICE_NAME).1` ]; \
                 	do \
                 		printf "."; \
                 		sleep 0.3;\
                 	done; \
                 	printf "\n";

log=docker logs $1$2 `docker ps -qf name=$(STACK_NAME)_$(SERVICE_NAME).1` 2>&1

build:
	$(MAKE) -C .. $(PROJECT_NAME)

iterate:
	$(call scale,0,-d)
	$(MAKE) build
	$(call scale,1,-d)
	$(call wait_for_service, start)
	$(MAKE) tail

restart:
	$(call scale,0,-d)
	$(call wait_for_service, stop, !)
	$(call scale,1,-d)
	$(call wait_for_service, start)

tail:
	$(call log,-f,$(args))

scale:
	$(call scale,$(n),$(args))

fmt:
	go fmt ./...

