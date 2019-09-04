
STACK_NAME ?= Inventory-Suite-Dev
SERVICE_NAME ?= inventory
PROJECT_NAME ?= inventory-service

.PHONY: clean-mongo

default: build

scale = docker service scale $(STACK_NAME)_$(SERVICE_NAME)=$1 $2

wait_for_service =	@printf "Waiting for $(SERVICE_NAME) service to$1..."; \
					while [  $2 -z $(get_id) ]; \
                 	do \
                 		printf "."; \
                 		sleep 0.3;\
                 	done; \
                 	printf "\n";

trap_ctrl_c = trap 'exit 0' INT;

get_id = `docker ps -qf name=$(STACK_NAME)_$(SERVICE_NAME).1`

log = docker logs $1$2 $(get_id) 2>&1

clean_mongo =   echo "Cleaning MongoDB test databases..."; \
				$(eval tmp := "$(shell mktemp).js") \
				echo ' \
					let count = 0; \
					let dbs = db.getMongo().getDBNames(); \
					for (var i in dbs) { \
						db = db.getMongo().getDB(dbs[i]); \
						if (db.getName().includes("_test_")) { \
							db.dropDatabase(); \
							count++; \
						} \
					} \
					print("Dropped " + count + " test databases"); \
				' | tr -d '\\' > $(tmp); \
				mongo --quiet $(tmp) || echo "Please install mongodb-clients to support auto database cleaning. (sudo apt install mongodb-clients)"; \
				rm -f $(tmp)

test =	echo "Go Testing..."; \
		go test ./... $1; \
     	$(call clean_mongo)

build:
	$(MAKE) -C .. $(PROJECT_NAME)

iterate:
	$(call scale,0,-d)
	$(MAKE) build
	# make sure it has stopped before we try and start it again
	$(call wait_for_service, stop, !)
	$(call scale,1,-d)
	$(call wait_for_service, start)
	$(MAKE) tail

restart:
	$(call scale,0,-d)
	$(call wait_for_service, stop, !)
	$(call scale,1,-d)
	$(call wait_for_service, start)

tail:
	$(trap_ctrl_c) $(call log,-f --tail 10,$(args))

stop:
	$(call scale,0,$(args))

start:
	$(call scale,1,$(args))

stop-d:
	$(call scale,0,-d)

start-d:
	$(call scale,1,-d)

wait-stop:
	$(call scale,0,-d)
	$(call wait_for_service, stop, !)

wait-start:
	$(call scale,1,-d)
	$(call wait_for_service, start)

scale:
	$(call scale,$(n),$(args))

fmt:
	go fmt ./...

test:
	@$(call test,$(args))

force-test:
	@$(call test,-count=1)

clean-mongo:
	@$(call clean_mongo)