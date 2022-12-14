DISCONTINUATION OF PROJECT. 

This project will no longer be maintained by Intel.

This project has been identified as having known security escapes.

Intel has ceased development and contributions including, but not limited to, maintenance, bug fixes, new releases, or updates, to this project.  

Intel no longer accepts patches to this project.
# Intel® Inventory Suite inventory-service
[![license](https://img.shields.io/badge/license-Apache%20v2.0-blue.svg)](LICENSE)

Inventory service is a microservice in the Intel® Inventory Suite that provides business context to raw RFID reads from Intel® RSP.
Some of the features are:
- Generates events based on raw RFID data. (arrival, move, and departed)
- Events are re-published to EdgeX Core data.
- Location history of a RFID tag.
- Data persistence in PostgreSQL.
- RESTful APIs using odata. 

# Depends on

- EdgeX Core-data
- Product-data-service 
- Rfid-alert-service
- Cloud-connector 

# Install and Deploy via Docker Container #

Intel® RSP Software Toolkit 

- [RSP Controller](https://github.com/intel/rsp-sw-toolkit-gw)
- [RSP MQTT Device Service](https://github.com/intel/rsp-sw-toolkit-im-suite-mqtt-device-service)

EdgeX and RSP MQTT Device Service should be running at this point.

### Installation ###

```
make build deploy
```

### API Documentation ###

Go to [https://editor.swagger.io](https://editor.swagger.io) and import inventory-service.yml file.
