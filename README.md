# Inventory service

See inventory-service.json for swagger documentation

## Linting
We use gometalinter.v2 for linting of code. The linter options are in a config file stored in the Go-Mongo-Docker-Build repository. You must clone this repository and pull latest prior to running the linter as follows:
```bash
gometalinter.v2 --vendor --deadline=120s --disable gotype --config=../../RSP-Inventory-Suite/ci-go-build-image/linter.json ./...
```
## Testing
In order to test your micro service using docker, compile your project and run docker-compose to orchestrate dependencies such as context sensing brokers (in & out), inventory-service and mapping-sku-service:

Compile and run your micro service in docker:

```bash
$ ./build.sh
$ sudo docker-compose up
```

### MongoDB Server
A mongodb server is required to run the unit tests. A quick way to get one up and running with docker:
```bash
$ mkdir -p ~/data
$ docker run -d -p 27017:27017 -v ~/data:/data/db mongo
```

## Swagger documentation
We use swagger to document the service details. See the following Wiki for details on using swagger to document the this service:
https://wiki.ith.intel.com/display/RSP/How+to+use+go-swagger

Use the following commands to generate and validate your swagger once you have instrumented the code:

 ### Generate Updated Swagger Doc
 Make sure you have goswagger installed (https://github.com/go-swagger/go-swagger): 
 
 `go get -u github.com/go-swagger/go-swagger/cmd/swagger`
 
  then run:
  
 `swagger generate spec -m -o inventory-service.json`
 
 #### Validate Generated Swagger Doc
 Run the following swagger command to validate the generated swagger JSON documentation file:
 
 `swagger validate ./inventory-service.json`
 
 Alternatively, the online swagger editor webpage (https://editor.swagger.io/) can also be used to validate the generated documentation. Just copy and paste the contents of JSON `inventory-service.json` onto the editing area of that webpage.
 
 
## Docker Image
The code pipeline will build the service and create the docker image and push it to: 

```280211473891.dkr.ecr.us-west-2.amazonaws.com/inventory-service```

Copyright 2018 Intel(R) Corporation, All rights reserved.
