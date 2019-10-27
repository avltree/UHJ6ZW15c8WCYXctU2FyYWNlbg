init:
	docker-compose up -d
	docker-compose run go bash -c 'go get && go install'

up:
	docker-compose up -d
	docker-compose exec go bash -c 'test -f go/bin/gwp-api || { echo "App executable does not exist, run make init"; exit 1; }'
	docker-compose exec -d go go/bin/gwp-api

down:
	docker-compose down
