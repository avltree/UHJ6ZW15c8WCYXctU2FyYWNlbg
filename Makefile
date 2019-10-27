init:
	docker-compose up -d
	docker-compose run go bash -c 'cd src/main && go get && go install'

up:
	docker-compose up -d
	docker-compose exec go bash -c 'test -f go/bin/main || { echo "App executable does not exist, run make init"; exit 1; }'
	docker-compose exec -d go go/bin/main

down:
	docker-compose down
