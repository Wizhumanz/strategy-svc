export PORT=8001

export REDISHOST_CM=35.240.201.212

export REDISPORT_CM=6379

export REDISPASS_CM=chartmaster676172378173812738712381238902

export PRINTLOGS=true

go build -o build/api .

build/api