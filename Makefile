ARGS = $(filter-out $@,$(MAKECMDGOALS))

build:
	docker-compose build

dev_up:
	docker-compose up -d

logs:
	docker-compose logs -f

down:
	docker-compose down

dep: 
	docker-compose exec app dep ensure -update

dep-add: 
	docker-compose exec app dep ensure -add $(ARGS)

dep-status: 
	make dep status

dev-local: 
	realize start

%:
	@: