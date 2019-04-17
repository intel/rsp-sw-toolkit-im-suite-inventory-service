#!/bin/bash
# inventory-service
set -e

printHelp() {
    echo "will run build the contents of the folder."
    echo "If --docker or -d is specified"
    echo "will run docker-compose up"
    echo
    echo "Usage: ./build.sh -d"
    echo
    echo "Options:"
    echo "  -d, --docker    Run the docker-compose up command"
    echo "  -db, --dockerb  Force a docker-compose rebuild"
    echo "  -h, --help      Show this help dialog"
    exit 0
}

# parameters
buildDocker=false
rebuildDocker=false
for var in "$@"; do
    case "${var}" in
        "-d" | "--docker"     ) buildDocker=true;;
        "-db" | "--dockerb"     ) rebuildDocker=true;;
        "-h" | "--help"      ) printHelp;;
    esac
done

echo -e "  \e[2mGo \e[0m\e[94mBuild(ing)...\e[0m"
#CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo
CGO_LDFLAGS+='-lstdc++ -lm' CGO_ENABLED=1 GOOS=linux go build -a --ldflags '-extldflags "-static" -v' -o ./inventory-service

if [[ "${rebuildDocker}" == true ]]; then
    echo -e "\e[94m rebuilding docker image..."
    sudo docker-compose up --build
elif [[ "${buildDocker}" == true ]]; then
    echo -e "\e[94m making docker image..."
    #    docker build --build-arg https_proxy=$https_proxy --build-arg http_proxy=$http_proxy  -t inventory .
    sudo docker-compose up
    #    docker run inventory
else
    echo -e "\e[2mNO Docker operation requested..."
fi

