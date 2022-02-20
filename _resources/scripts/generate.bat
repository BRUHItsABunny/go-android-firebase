@echo off
docker build -t gaf_protos .
docker run --name="gaf_protos_run" gaf_protos
docker cp gaf_protos_run:proto/. .
docker rm gaf_protos_run
docker rmi gaf_protos
echo "BAT done"